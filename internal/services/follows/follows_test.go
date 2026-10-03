package follows_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/weeb-vip/user-service/internal/entities"
	"github.com/weeb-vip/user-service/internal/services/follows"
	followmodels "github.com/weeb-vip/user-service/internal/services/follows/models"
	usermodels "github.com/weeb-vip/user-service/internal/services/users/models"
	"github.com/weeb-vip/user-service/mocks"
)

const (
	alice = "user_alice"
	bob   = "user_bob"
)

type fixture struct {
	repo   *mocks.MockFollowsRepository
	users  *mocks.MockUserLookup
	events *mocks.MockEventWriter
	svc    follows.Follows
	// written collects every event handed to the EventWriter.
	written []follows.Event
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	f := &fixture{
		repo:   mocks.NewMockFollowsRepository(ctrl),
		users:  mocks.NewMockUserLookup(ctrl),
		events: mocks.NewMockEventWriter(ctrl),
	}
	f.svc = follows.New(f.repo, f.users, f.events)

	// The service's transaction is the mock calling back with a nil tx, which
	// the repo and event mocks accept; what matters is that both run inside it.
	f.repo.EXPECT().Transaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(tx *gorm.DB) error) error { return fn(nil) }).
		AnyTimes()
	f.events.EXPECT().Write(gomock.Any(), gomock.Any(), follows.Subject, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ *gorm.DB, _ string, id string, payload any) error {
			// Round-trip through JSON: that is what the outbox stores and the
			// consumer reads, so assert on that shape rather than the struct.
			body, err := json.Marshal(payload)
			require.NoError(t, err)
			var ev follows.Event
			require.NoError(t, json.Unmarshal(body, &ev))
			assert.Equal(t, id, ev.ID, "outbox id and payload id must match")
			f.written = append(f.written, ev)

			return nil
		}).
		AnyTimes()

	return f
}

func user(id string, approval bool) *usermodels.User {
	u := &usermodels.User{FollowApprovalRequired: approval}
	u.ID = id

	return u
}

func edge(status followmodels.Status) *followmodels.UserFollow {
	return &followmodels.UserFollow{FollowerID: alice, FolloweeID: bob, Status: status, CreatedAt: time.Now()}
}

func serviceCode(t *testing.T, err error) string {
	t.Helper()
	var svcErr *entities.ServiceError
	require.ErrorAs(t, err, &svcErr)

	return svcErr.Code
}

func TestFollow(t *testing.T) {
	t.Run("open account is followed immediately and announces follow_accepted", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, false), nil)
		f.repo.EXPECT().Get(gomock.Any(), alice, bob).Return(nil, nil)
		f.repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ *gorm.DB, follow *followmodels.UserFollow) error {
				assert.Equal(t, alice, follow.FollowerID)
				assert.Equal(t, bob, follow.FolloweeID)
				assert.Equal(t, followmodels.StatusAccepted, follow.Status)
				assert.NotNil(t, follow.AcceptedAt)

				return nil
			})

		status, err := f.svc.Follow(context.Background(), alice, bob)
		require.NoError(t, err)
		assert.Equal(t, follows.ViewerFollowing, status)

		require.Len(t, f.written, 1)
		ev := f.written[0]
		assert.Equal(t, follows.EventFollowAccepted, ev.Type)
		assert.Equal(t, alice, ev.FollowerID)
		assert.Equal(t, bob, ev.FolloweeID)
		assert.True(t, ev.AutoAccepted)
		assert.NotEmpty(t, ev.ID)
		assert.False(t, ev.OccurredAt.IsZero())
	})

	t.Run("approval-required account gets a pending request and follow_requested", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, true), nil)
		f.repo.EXPECT().Get(gomock.Any(), alice, bob).Return(nil, nil)
		f.repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ *gorm.DB, follow *followmodels.UserFollow) error {
				assert.Equal(t, followmodels.StatusPending, follow.Status)
				assert.Nil(t, follow.AcceptedAt)

				return nil
			})

		status, err := f.svc.Follow(context.Background(), alice, bob)
		require.NoError(t, err)
		assert.Equal(t, follows.ViewerRequested, status)

		require.Len(t, f.written, 1)
		assert.Equal(t, follows.EventFollowRequested, f.written[0].Type)
		assert.False(t, f.written[0].AutoAccepted)
	})

	t.Run("self follow is rejected before any lookup", func(t *testing.T) {
		f := newFixture(t)

		_, err := f.svc.Follow(context.Background(), alice, alice)
		assert.Equal(t, follows.CodeSelfFollow, serviceCode(t, err))
		assert.Empty(t, f.written)
	})

	t.Run("unknown target is USER_NOT_FOUND", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(&usermodels.User{}, nil)

		_, err := f.svc.Follow(context.Background(), alice, bob)
		assert.Equal(t, follows.CodeUserNotFound, serviceCode(t, err))
		assert.Empty(t, f.written)
	})

	t.Run("repeat follow is a no-op reporting the current state", func(t *testing.T) {
		for _, tc := range []struct {
			existing followmodels.Status
			want     follows.ViewerStatus
		}{
			{followmodels.StatusAccepted, follows.ViewerFollowing},
			{followmodels.StatusPending, follows.ViewerRequested},
		} {
			f := newFixture(t)
			f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, true), nil)
			f.repo.EXPECT().Get(gomock.Any(), alice, bob).Return(edge(tc.existing), nil)

			status, err := f.svc.Follow(context.Background(), alice, bob)
			require.NoError(t, err)
			assert.Equal(t, tc.want, status)
			assert.Empty(t, f.written, "no event for a no-op")
		}
	})

	t.Run("a failing event write fails the follow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockFollowsRepository(ctrl)
		users := mocks.NewMockUserLookup(ctrl)
		events := mocks.NewMockEventWriter(ctrl)
		svc := follows.New(repo, users, events)

		boom := errors.New("outbox down")
		users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, false), nil)
		repo.EXPECT().Get(gomock.Any(), alice, bob).Return(nil, nil)
		repo.EXPECT().Transaction(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, fn func(tx *gorm.DB) error) error { return fn(nil) })
		repo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		events.EXPECT().Write(gomock.Any(), gomock.Any(), follows.Subject, gomock.Any(), gomock.Any()).Return(boom)

		_, err := svc.Follow(context.Background(), alice, bob)
		require.ErrorIs(t, err, boom)
	})
}

func TestUnfollow(t *testing.T) {
	t.Run("removes an accepted edge and announces unfollowed", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Get(gomock.Any(), alice, bob).Return(edge(followmodels.StatusAccepted), nil)
		f.repo.EXPECT().Delete(gomock.Any(), gomock.Any(), alice, bob, followmodels.Status("")).Return(int64(1), nil)

		removed, err := f.svc.Unfollow(context.Background(), alice, bob)
		require.NoError(t, err)
		assert.True(t, removed)
		require.Len(t, f.written, 1)
		assert.Equal(t, follows.EventUnfollowed, f.written[0].Type)
		assert.Equal(t, "accepted", f.written[0].PreviousStatus)
	})

	t.Run("withdrawing a pending request records the previous status", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Get(gomock.Any(), alice, bob).Return(edge(followmodels.StatusPending), nil)
		f.repo.EXPECT().Delete(gomock.Any(), gomock.Any(), alice, bob, followmodels.Status("")).Return(int64(1), nil)

		removed, err := f.svc.Unfollow(context.Background(), alice, bob)
		require.NoError(t, err)
		assert.True(t, removed)
		assert.Equal(t, "pending", f.written[0].PreviousStatus)
	})

	t.Run("no edge is a no-op with no event", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Get(gomock.Any(), alice, bob).Return(nil, nil)

		removed, err := f.svc.Unfollow(context.Background(), alice, bob)
		require.NoError(t, err)
		assert.False(t, removed)
		assert.Empty(t, f.written)
	})
}

func TestAcceptDeclineRemove(t *testing.T) {
	t.Run("accept flips a pending request and announces a non-auto follow_accepted", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Accept(gomock.Any(), gomock.Any(), alice, bob, gomock.Any()).Return(int64(1), nil)

		require.NoError(t, f.svc.Accept(context.Background(), bob, alice))
		require.Len(t, f.written, 1)
		assert.Equal(t, follows.EventFollowAccepted, f.written[0].Type)
		assert.Equal(t, alice, f.written[0].FollowerID)
		assert.Equal(t, bob, f.written[0].FolloweeID)
		assert.False(t, f.written[0].AutoAccepted)
	})

	t.Run("accept with no pending request is FOLLOW_REQUEST_NOT_FOUND", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Accept(gomock.Any(), gomock.Any(), alice, bob, gomock.Any()).Return(int64(0), nil)

		err := f.svc.Accept(context.Background(), bob, alice)
		assert.Equal(t, follows.CodeRequestNotFound, serviceCode(t, err))
		assert.Empty(t, f.written)
	})

	t.Run("decline deletes only a pending edge", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Delete(gomock.Any(), gomock.Any(), alice, bob, followmodels.StatusPending).Return(int64(1), nil)

		require.NoError(t, f.svc.Decline(context.Background(), bob, alice))
		require.Len(t, f.written, 1)
		assert.Equal(t, follows.EventFollowDeclined, f.written[0].Type)
	})

	t.Run("decline with nothing pending is FOLLOW_REQUEST_NOT_FOUND", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Delete(gomock.Any(), gomock.Any(), alice, bob, followmodels.StatusPending).Return(int64(0), nil)

		err := f.svc.Decline(context.Background(), bob, alice)
		assert.Equal(t, follows.CodeRequestNotFound, serviceCode(t, err))
	})

	t.Run("remove follower deletes only an accepted edge", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Delete(gomock.Any(), gomock.Any(), alice, bob, followmodels.StatusAccepted).Return(int64(1), nil)

		require.NoError(t, f.svc.RemoveFollower(context.Background(), bob, alice))
		require.Len(t, f.written, 1)
		assert.Equal(t, follows.EventFollowerRemoved, f.written[0].Type)
		assert.Equal(t, "accepted", f.written[0].PreviousStatus)
	})

	t.Run("remove follower with no accepted edge is FOLLOW_NOT_FOUND", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Delete(gomock.Any(), gomock.Any(), alice, bob, followmodels.StatusAccepted).Return(int64(0), nil)

		err := f.svc.RemoveFollower(context.Background(), bob, alice)
		assert.Equal(t, follows.CodeFollowNotFound, serviceCode(t, err))
	})
}

func TestViewerStatus(t *testing.T) {
	viewerID := alice
	cases := []struct {
		name   string
		viewer *string
		edge   *followmodels.UserFollow
		want   follows.ViewerStatus
	}{
		{"anonymous", nil, nil, follows.ViewerNone},
		{"no edge", &viewerID, nil, follows.ViewerNone},
		{"pending", &viewerID, edge(followmodels.StatusPending), follows.ViewerRequested},
		{"accepted", &viewerID, edge(followmodels.StatusAccepted), follows.ViewerFollowing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.viewer != nil {
				f.repo.EXPECT().Get(gomock.Any(), *tc.viewer, bob).Return(tc.edge, nil)
			}

			got, err := f.svc.ViewerStatus(context.Background(), tc.viewer, bob)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("self", func(t *testing.T) {
		f := newFixture(t)
		got, err := f.svc.ViewerStatus(context.Background(), &viewerID, alice)
		require.NoError(t, err)
		assert.Equal(t, follows.ViewerSelf, got)
	})
}

func TestFollowersVisibility(t *testing.T) {
	page := []*usermodels.User{user(alice, false)}

	t.Run("open account lists to anyone", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, false), nil)
		f.repo.EXPECT().ListFollowers(gomock.Any(), bob, followmodels.StatusAccepted, 1, 20).Return(page, int64(1), nil)

		got, err := f.svc.Followers(context.Background(), nil, bob, 1, 20)
		require.NoError(t, err)
		assert.Len(t, got.Users, 1)
		assert.EqualValues(t, 1, got.Total)
	})

	t.Run("approval-required account hides names from an anonymous viewer but keeps the total", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, true), nil)
		f.repo.EXPECT().Counts(gomock.Any(), bob).Return(followmodels.Counts{Followers: 7, Following: 3}, nil)

		got, err := f.svc.Followers(context.Background(), nil, bob, 1, 20)
		require.NoError(t, err)
		assert.Empty(t, got.Users)
		assert.EqualValues(t, 7, got.Total)
	})

	t.Run("approval-required account hides names from a non-follower", func(t *testing.T) {
		f := newFixture(t)
		viewerID := "user_carol"
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, true), nil)
		f.repo.EXPECT().Get(gomock.Any(), viewerID, bob).Return(edge(followmodels.StatusPending), nil)
		f.repo.EXPECT().Counts(gomock.Any(), bob).Return(followmodels.Counts{Following: 3}, nil)

		got, err := f.svc.Following(context.Background(), &viewerID, bob, 1, 20)
		require.NoError(t, err)
		assert.Empty(t, got.Users)
		assert.EqualValues(t, 3, got.Total)
	})

	t.Run("approval-required account lists to an accepted follower", func(t *testing.T) {
		f := newFixture(t)
		viewerID := alice
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, true), nil)
		f.repo.EXPECT().Get(gomock.Any(), viewerID, bob).Return(edge(followmodels.StatusAccepted), nil)
		f.repo.EXPECT().ListFollowers(gomock.Any(), bob, followmodels.StatusAccepted, 2, 10).Return(page, int64(11), nil)

		got, err := f.svc.Followers(context.Background(), &viewerID, bob, 2, 10)
		require.NoError(t, err)
		assert.Len(t, got.Users, 1)
		assert.EqualValues(t, 11, got.Total)
	})

	t.Run("approval-required account lists to its owner", func(t *testing.T) {
		f := newFixture(t)
		viewerID := bob
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(user(bob, true), nil)
		f.repo.EXPECT().ListFollowing(gomock.Any(), bob, followmodels.StatusAccepted, 1, 20).Return(nil, int64(0), nil)

		got, err := f.svc.Following(context.Background(), &viewerID, bob, 1, 20)
		require.NoError(t, err)
		assert.NotNil(t, got.Users, "an empty page is an empty list, not null")
		assert.Empty(t, got.Users)
	})

	t.Run("unknown user is USER_NOT_FOUND", func(t *testing.T) {
		f := newFixture(t)
		f.users.EXPECT().GetUserDetails(gomock.Any(), bob).Return(nil, nil)

		_, err := f.svc.Followers(context.Background(), nil, bob, 1, 20)
		assert.Equal(t, follows.CodeUserNotFound, serviceCode(t, err))
	})
}

func TestRequestsCountsFollowerIDs(t *testing.T) {
	t.Run("requests are the pending followers of the owner", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().ListFollowers(gomock.Any(), bob, followmodels.StatusPending, 1, 20).
			Return([]*usermodels.User{user(alice, false)}, int64(1), nil)

		got, err := f.svc.Requests(context.Background(), bob, 1, 20)
		require.NoError(t, err)
		assert.Len(t, got.Users, 1)
	})

	t.Run("counts pass through", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Counts(gomock.Any(), bob).Return(followmodels.Counts{Followers: 2, Following: 5}, nil)

		got, err := f.svc.Counts(context.Background(), bob)
		require.NoError(t, err)
		assert.Equal(t, followmodels.Counts{Followers: 2, Following: 5}, got)
	})

	t.Run("follower ids never come back nil", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().FollowerIDs(gomock.Any(), bob, "", 500).Return(nil, nil)

		got, err := f.svc.FollowerIDs(context.Background(), bob, "", 500)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}
