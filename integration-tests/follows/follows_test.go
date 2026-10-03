//go:build integration

package follows

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenFollow(t *testing.T) {
	alice := newUser(t, "alice")
	bob := newUser(t, "bob")

	// Before: nothing between them, and bob is visible to alice by id.
	bobSeenByAlice := publicUserByID(t, alice, bob)
	assert.Equal(t, "NONE", bobSeenByAlice.ViewerFollowStatus)
	assert.Equal(t, 0, bobSeenByAlice.FollowerCount)

	assert.Equal(t, "FOLLOWING", follow(t, alice, bob))
	assert.Equal(t, "FOLLOWING", follow(t, alice, bob), "repeat follow reports the same state")

	bobSeenByAlice = publicUserByID(t, alice, bob)
	assert.Equal(t, "FOLLOWING", bobSeenByAlice.ViewerFollowStatus)
	assert.Equal(t, 1, bobSeenByAlice.FollowerCount)
	assert.Equal(t, 0, bobSeenByAlice.FollowingCount)

	assert.Equal(t, "SELF", publicUserByID(t, bob, bob).ViewerFollowStatus)
	assert.Equal(t, "NONE", publicUserByID(t, "", bob).ViewerFollowStatus, "anonymous viewer")

	// Lists are public for an open account, even anonymously.
	bobFollowers := followers(t, "", bob)
	assert.Equal(t, 1, bobFollowers.Total)
	require.Len(t, bobFollowers.Users, 1)
	assert.Equal(t, alice, bobFollowers.Users[0].ID)

	aliceFollowing := following(t, "", alice)
	assert.Equal(t, 1, aliceFollowing.Total)
	require.Len(t, aliceFollowing.Users, 1)
	assert.Equal(t, bob, aliceFollowing.Users[0].ID)

	// Fan-out paging sees alice.
	var idsOut struct{ FollowerIDs []string }
	mustData(t, "", `query($id: ID!) { followerIDs(userID: $id, limit: 100) }`, map[string]any{"id": bob}, &idsOut)
	assert.Equal(t, []string{alice}, idsOut.FollowerIDs)

	// Unfollow, twice: the second finds nothing.
	var un struct{ Unfollow bool }
	mustData(t, alice, `mutation($id: ID!) { unfollow(userID: $id) }`, map[string]any{"id": bob}, &un)
	assert.True(t, un.Unfollow)
	mustData(t, alice, `mutation($id: ID!) { unfollow(userID: $id) }`, map[string]any{"id": bob}, &un)
	assert.False(t, un.Unfollow)

	assert.Equal(t, 0, publicUserByID(t, "", bob).FollowerCount)

	// Exactly one event per state change, in order, and none for the no-ops.
	assert.Equal(t, []string{"follow_accepted", "unfollowed"}, outboxTypes(t, alice, bob))
}

func TestApprovalRequiredFollow(t *testing.T) {
	alice := newUser(t, "alice")
	bob := newUser(t, "bob")
	carol := newUser(t, "carol")
	setApproval(t, bob, true)

	assert.Equal(t, "REQUESTED", follow(t, alice, bob))
	assert.Equal(t, "REQUESTED", publicUserByID(t, alice, bob).ViewerFollowStatus)
	assert.Equal(t, 0, publicUserByID(t, alice, bob).FollowerCount, "pending is not a follower")

	// Only bob sees the request.
	var reqs struct{ FollowRequests page }
	mustData(t, bob, `query { followRequests(page: 1, limit: 10) { total users { id } } }`, nil, &reqs)
	assert.Equal(t, 1, reqs.FollowRequests.Total)
	require.Len(t, reqs.FollowRequests.Users, 1)
	assert.Equal(t, alice, reqs.FollowRequests.Users[0].ID)

	resp := query(t, "", `query { followRequests(page: 1, limit: 10) { total } }`, nil)
	assert.NotEmpty(t, resp.Errors, "followRequests needs a signed-in user")

	// Carol cannot accept on bob's behalf.
	resp = query(t, carol, `mutation($id: ID!) { acceptFollowRequest(followerID: $id) }`, map[string]any{"id": alice})
	assert.Equal(t, "FOLLOW_REQUEST_NOT_FOUND", resp.code())

	var acc struct{ AcceptFollowRequest bool }
	mustData(t, bob, `mutation($id: ID!) { acceptFollowRequest(followerID: $id) }`, map[string]any{"id": alice}, &acc)
	assert.True(t, acc.AcceptFollowRequest)
	assert.Equal(t, "FOLLOWING", publicUserByID(t, alice, bob).ViewerFollowStatus)
	assert.Equal(t, 1, publicUserByID(t, "", bob).FollowerCount)

	// Visibility: bob's lists are hidden from carol and from anonymous
	// viewers, visible to alice (accepted) and bob. Totals stay truthful.
	for _, viewer := range []string{"", carol} {
		p := followers(t, viewer, bob)
		assert.Equal(t, 1, p.Total)
		assert.Empty(t, p.Users, "viewer %q must not see names", viewer)
	}
	for _, viewer := range []string{alice, bob} {
		p := followers(t, viewer, bob)
		assert.Equal(t, 1, p.Total)
		require.Len(t, p.Users, 1, "viewer %q sees names", viewer)
		assert.Equal(t, alice, p.Users[0].ID)
	}

	// Carol asks, bob declines; carol can ask again afterwards.
	assert.Equal(t, "REQUESTED", follow(t, carol, bob))
	var dec struct{ DeclineFollowRequest bool }
	mustData(t, bob, `mutation($id: ID!) { declineFollowRequest(followerID: $id) }`, map[string]any{"id": carol}, &dec)
	assert.True(t, dec.DeclineFollowRequest)
	assert.Equal(t, "NONE", publicUserByID(t, carol, bob).ViewerFollowStatus)
	resp = query(t, bob, `mutation($id: ID!) { declineFollowRequest(followerID: $id) }`, map[string]any{"id": carol})
	assert.Equal(t, "FOLLOW_REQUEST_NOT_FOUND", resp.code())

	// Bob removes alice.
	var rm struct{ RemoveFollower bool }
	mustData(t, bob, `mutation($id: ID!) { removeFollower(followerID: $id) }`, map[string]any{"id": alice}, &rm)
	assert.True(t, rm.RemoveFollower)
	assert.Equal(t, "NONE", publicUserByID(t, alice, bob).ViewerFollowStatus)
	resp = query(t, bob, `mutation($id: ID!) { removeFollower(followerID: $id) }`, map[string]any{"id": alice})
	assert.Equal(t, "FOLLOW_NOT_FOUND", resp.code())

	assert.Equal(t, []string{"follow_requested", "follow_accepted", "follower_removed"}, outboxTypes(t, alice, bob))
	assert.Equal(t, []string{"follow_requested", "follow_declined"}, outboxTypes(t, carol, bob))
}

func TestFollowErrors(t *testing.T) {
	alice := newUser(t, "alice")

	resp := query(t, alice, `mutation($id: ID!) { follow(userID: $id) }`, map[string]any{"id": alice})
	assert.Equal(t, "FOLLOW_SELF", resp.code())

	resp = query(t, alice, `mutation($id: ID!) { follow(userID: $id) }`, map[string]any{"id": "user_does_not_exist"})
	assert.Equal(t, "USER_NOT_FOUND", resp.code())

	resp = query(t, "", `mutation($id: ID!) { follow(userID: $id) }`, map[string]any{"id": alice})
	assert.Equal(t, "UNAUTHORIZED", resp.code())

	assert.Empty(t, outboxTypes(t, alice, alice))
}

func TestPublicUserEntity(t *testing.T) {
	alice := newUser(t, "alice")
	bob := newUser(t, "bob")
	follow(t, alice, bob)

	// What the router sends when another subgraph returns PublicUser{id}.
	var out struct {
		Entities []json.RawMessage `json:"_entities"`
	}
	mustData(t, alice, `query($reps: [_Any!]!) {
		_entities(representations: $reps) {
			... on PublicUser { id username followerCount viewerFollowStatus }
		}
	}`, map[string]any{"reps": []map[string]any{{"__typename": "PublicUser", "id": bob}, {"__typename": "PublicUser", "id": "user_missing"}}}, &out)
	require.Len(t, out.Entities, 2)

	var resolved publicUser
	require.NoError(t, json.Unmarshal(out.Entities[0], &resolved))
	assert.Equal(t, bob, resolved.ID)
	assert.Equal(t, bob, resolved.Username)
	assert.Equal(t, 1, resolved.FollowerCount)
	assert.Equal(t, "FOLLOWING", resolved.ViewerFollowStatus)
	assert.Equal(t, "null", string(out.Entities[1]), "an unknown id resolves to null, not an error")

	var byID struct{ PublicUserByID *publicUser }
	mustData(t, "", `query { publicUserByID(id: "user_missing") { id } }`, nil, &byID)
	assert.Nil(t, byID.PublicUserByID)
}
