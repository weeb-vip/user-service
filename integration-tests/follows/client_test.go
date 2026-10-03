//go:build integration

package follows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type gqlError struct {
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors"`
}

// code returns the envelope code of the first error, or "".
func (r *gqlResponse) code() string {
	if len(r.Errors) == 0 {
		return ""
	}
	c, _ := r.Errors[0].Extensions["code"].(string)

	return c
}

// query runs an operation as userID; an empty userID is an anonymous call.
func query(t *testing.T, userID string, operation string, variables map[string]any) *gqlResponse {
	t.Helper()
	body, err := json.Marshal(map[string]any{"query": operation, "variables": variables})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/graphql", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("x-user-id", userID)
		req.Header.Set("x-token-purpose", "test")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var out gqlResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))

	return &out
}

// mustData runs an operation and fails the test on any GraphQL error.
func mustData(t *testing.T, userID, operation string, variables map[string]any, into any) {
	t.Helper()
	resp := query(t, userID, operation, variables)
	require.Empty(t, resp.Errors, "unexpected GraphQL errors: %+v", resp.Errors)
	require.NoError(t, json.Unmarshal(resp.Data, into))
}

// newUser creates a user through the API as that user, and removes it after.
func newUser(t *testing.T, label string) string {
	t.Helper()
	id := fmt.Sprintf("user_%s_%d", label, time.Now().UnixNano())
	var out struct {
		CreatUser struct{ ID string }
	}
	mustData(t, id, `mutation($input: CreateUserInput!) { CreatUser(input: $input) { id } }`, map[string]any{
		"input": map[string]any{"id": id, "firstname": label, "lastname": "Test", "username": id, "language": "EN"},
	}, &out)
	require.Equal(t, id, out.CreatUser.ID)
	t.Cleanup(func() {
		database.Exec("DELETE FROM user_follows WHERE follower_id = ? OR followee_id = ?", id, id)
		database.Exec("DELETE FROM outbox_events WHERE payload->>'follower_id' = ? OR payload->>'followee_id' = ?", id, id)
		database.Exec("DELETE FROM users WHERE id = ?", id)
	})

	return id
}

func setApproval(t *testing.T, userID string, required bool) {
	t.Helper()
	var out struct {
		UpdateUserDetails struct{ FollowApprovalRequired bool }
	}
	mustData(t, userID, `mutation($input: UpdateUserInput!) { UpdateUserDetails(input: $input) { followApprovalRequired } }`,
		map[string]any{"input": map[string]any{"followApprovalRequired": required}}, &out)
	require.Equal(t, required, out.UpdateUserDetails.FollowApprovalRequired)
}

type publicUser struct {
	ID                     string `json:"id"`
	Username               string `json:"username"`
	FollowApprovalRequired bool   `json:"followApprovalRequired"`
	FollowerCount          int    `json:"followerCount"`
	FollowingCount         int    `json:"followingCount"`
	ViewerFollowStatus     string `json:"viewerFollowStatus"`
}

type page struct {
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
	Total int          `json:"total"`
	Users []publicUser `json:"users"`
}

func follow(t *testing.T, viewer, target string) string {
	t.Helper()
	var out struct{ Follow string }
	mustData(t, viewer, `mutation($id: ID!) { follow(userID: $id) }`, map[string]any{"id": target}, &out)

	return out.Follow
}

func publicUserByID(t *testing.T, viewer, id string) publicUser {
	t.Helper()
	var out struct{ PublicUserByID publicUser }
	mustData(t, viewer, `query($id: ID!) { publicUserByID(id: $id) { id username followApprovalRequired followerCount followingCount viewerFollowStatus } }`,
		map[string]any{"id": id}, &out)

	return out.PublicUserByID
}

func followers(t *testing.T, viewer, id string) page {
	t.Helper()
	var out struct{ Followers page }
	mustData(t, viewer, `query($id: ID!) { followers(userID: $id, page: 1, limit: 10) { page limit total users { id username } } }`,
		map[string]any{"id": id}, &out)

	return out.Followers
}

func following(t *testing.T, viewer, id string) page {
	t.Helper()
	var out struct{ Following page }
	mustData(t, viewer, `query($id: ID!) { following(userID: $id, page: 1, limit: 10) { page limit total users { id username } } }`,
		map[string]any{"id": id}, &out)

	return out.Following
}

// outboxTypes returns the event types written for a follower -> followee pair,
// oldest first. Reading the table directly is deliberate: the relay is tested
// in go-outbox-lib, and what this service owes is the row.
func outboxTypes(t *testing.T, follower, followee string) []string {
	t.Helper()
	var types []string
	require.NoError(t, database.Raw(`
		SELECT payload->>'type' FROM outbox_events
		WHERE subject = 'user-follow' AND payload->>'follower_id' = ? AND payload->>'followee_id' = ?
		ORDER BY created_at, id`, follower, followee).Scan(&types).Error)

	return types
}
