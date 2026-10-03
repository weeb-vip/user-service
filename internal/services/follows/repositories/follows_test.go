package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/weeb-vip/user-service/internal/db"
	followmodels "github.com/weeb-vip/user-service/internal/services/follows/models"
	"github.com/weeb-vip/user-service/internal/services/follows/repositories"
	usermodels "github.com/weeb-vip/user-service/internal/services/users/models"
)

// These run against the real database the rest of the package tests use
// (DBHOST etc.), because the queries are the point: the join, the keyset page
// and the status filters are what a mock could not check.

func seedUser(t *testing.T, database *gorm.DB, id string) {
	t.Helper()
	user := usermodels.User{FirstName: "f", LastName: "l", Language: "EN"}
	user.ID = id
	username := id
	user.Username = &username
	require.NoError(t, database.Create(&user).Error)
	t.Cleanup(func() {
		database.Exec("DELETE FROM user_follows WHERE follower_id = ? OR followee_id = ?", id, id)
		database.Exec("DELETE FROM users WHERE id = ?", id)
	})
}

func seedFollow(t *testing.T, database *gorm.DB, follower, followee string, status followmodels.Status, at time.Time) {
	t.Helper()
	follow := followmodels.UserFollow{FollowerID: follower, FolloweeID: followee, Status: status, CreatedAt: at}
	if status == followmodels.StatusAccepted {
		follow.AcceptedAt = &at
	}
	require.NoError(t, database.Create(&follow).Error)
}

func ids(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("user_%s_%02d_%d", prefix, i, time.Now().UnixNano()%1_000_000)
	}

	return out
}

func TestFollowsRepository(t *testing.T) {
	ctx := context.Background()
	database := db.GetDBService().GetDB()
	repo := repositories.NewFollowsRepository()

	t.Run("create, get, accept, delete with status guard", func(t *testing.T) {
		u := ids("cgad", 2)
		seedUser(t, database, u[0])
		seedUser(t, database, u[1])

		require.NoError(t, repo.Transaction(ctx, func(tx *gorm.DB) error {
			return repo.Create(ctx, tx, &followmodels.UserFollow{
				FollowerID: u[0], FolloweeID: u[1], Status: followmodels.StatusPending, CreatedAt: time.Now(),
			})
		}))

		got, err := repo.Get(ctx, u[0], u[1])
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, followmodels.StatusPending, got.Status)
		assert.Nil(t, got.AcceptedAt)

		missing, err := repo.Get(ctx, u[1], u[0])
		require.NoError(t, err)
		assert.Nil(t, missing, "edges are directed")

		// Deleting with the wrong status guard touches nothing.
		var n int64
		require.NoError(t, repo.Transaction(ctx, func(tx *gorm.DB) error {
			var err error
			n, err = repo.Delete(ctx, tx, u[0], u[1], followmodels.StatusAccepted)

			return err
		}))
		assert.EqualValues(t, 0, n)

		require.NoError(t, repo.Transaction(ctx, func(tx *gorm.DB) error {
			var err error
			n, err = repo.Accept(ctx, tx, u[0], u[1], time.Now())

			return err
		}))
		assert.EqualValues(t, 1, n)
		got, err = repo.Get(ctx, u[0], u[1])
		require.NoError(t, err)
		assert.Equal(t, followmodels.StatusAccepted, got.Status)
		assert.NotNil(t, got.AcceptedAt)

		// Accepting again changes nothing: it only moves pending rows.
		require.NoError(t, repo.Transaction(ctx, func(tx *gorm.DB) error {
			var err error
			n, err = repo.Accept(ctx, tx, u[0], u[1], time.Now())

			return err
		}))
		assert.EqualValues(t, 0, n)

		require.NoError(t, repo.Transaction(ctx, func(tx *gorm.DB) error {
			var err error
			n, err = repo.Delete(ctx, tx, u[0], u[1], "")

			return err
		}))
		assert.EqualValues(t, 1, n)
	})

	t.Run("a rolled back transaction leaves no edge", func(t *testing.T) {
		u := ids("rb", 2)
		seedUser(t, database, u[0])
		seedUser(t, database, u[1])

		err := repo.Transaction(ctx, func(tx *gorm.DB) error {
			if err := repo.Create(ctx, tx, &followmodels.UserFollow{
				FollowerID: u[0], FolloweeID: u[1], Status: followmodels.StatusAccepted, CreatedAt: time.Now(),
			}); err != nil {
				return err
			}

			return fmt.Errorf("abort")
		})
		require.Error(t, err)

		got, err := repo.Get(ctx, u[0], u[1])
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("database constraints reject self follows and unknown statuses", func(t *testing.T) {
		u := ids("ck", 2)
		seedUser(t, database, u[0])
		seedUser(t, database, u[1])

		err := repo.Transaction(ctx, func(tx *gorm.DB) error {
			return repo.Create(ctx, tx, &followmodels.UserFollow{FollowerID: u[0], FolloweeID: u[0], Status: followmodels.StatusAccepted, CreatedAt: time.Now()})
		})
		assert.Error(t, err)

		err = repo.Transaction(ctx, func(tx *gorm.DB) error {
			return repo.Create(ctx, tx, &followmodels.UserFollow{FollowerID: u[0], FolloweeID: u[1], Status: "blocked", CreatedAt: time.Now()})
		})
		assert.Error(t, err)
	})

	t.Run("lists, counts and follower ids", func(t *testing.T) {
		u := ids("ls", 5)
		for _, id := range u {
			seedUser(t, database, id)
		}
		target := u[0]
		base := time.Now().Add(-time.Hour)
		// u1, u2, u3 follow target (accepted), u4 is pending; target follows u1.
		seedFollow(t, database, u[1], target, followmodels.StatusAccepted, base.Add(1*time.Minute))
		seedFollow(t, database, u[2], target, followmodels.StatusAccepted, base.Add(2*time.Minute))
		seedFollow(t, database, u[3], target, followmodels.StatusAccepted, base.Add(3*time.Minute))
		seedFollow(t, database, u[4], target, followmodels.StatusPending, base.Add(4*time.Minute))
		seedFollow(t, database, target, u[1], followmodels.StatusAccepted, base)

		followers, total, err := repo.ListFollowers(ctx, target, followmodels.StatusAccepted, 1, 2)
		require.NoError(t, err)
		assert.EqualValues(t, 3, total)
		require.Len(t, followers, 2)
		assert.Equal(t, u[3], followers[0].ID, "newest follow first")
		assert.Equal(t, u[2], followers[1].ID)

		followers, total, err = repo.ListFollowers(ctx, target, followmodels.StatusAccepted, 2, 2)
		require.NoError(t, err)
		assert.EqualValues(t, 3, total)
		require.Len(t, followers, 1)
		assert.Equal(t, u[1], followers[0].ID)

		pending, total, err := repo.ListFollowers(ctx, target, followmodels.StatusPending, 1, 10)
		require.NoError(t, err)
		assert.EqualValues(t, 1, total)
		require.Len(t, pending, 1)
		assert.Equal(t, u[4], pending[0].ID)

		following, total, err := repo.ListFollowing(ctx, target, followmodels.StatusAccepted, 1, 10)
		require.NoError(t, err)
		assert.EqualValues(t, 1, total)
		require.Len(t, following, 1)
		assert.Equal(t, u[1], following[0].ID)

		counts, err := repo.Counts(ctx, target)
		require.NoError(t, err)
		assert.Equal(t, followmodels.Counts{Followers: 3, Following: 1}, counts, "pending edges are not counted")

		// Keyset paging walks every accepted follower exactly once, in id order.
		var all []string
		after := ""
		for {
			page, err := repo.FollowerIDs(ctx, target, after, 2)
			require.NoError(t, err)
			if len(page) == 0 {
				break
			}
			all = append(all, page...)
			after = page[len(page)-1]
		}
		assert.ElementsMatch(t, []string{u[1], u[2], u[3]}, all)
		assert.IsNonDecreasing(t, all)
	})
}
