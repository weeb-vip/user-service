//go:build integration

package integration_tests

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// GraphQLClient handles GraphQL requests for integration tests
type GraphQLClient struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

// NewGraphQLClient creates a new GraphQL client
func NewGraphQLClient(baseURL string) *GraphQLClient {
	return &GraphQLClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

// SetAuthToken sets the authentication token for requests
func (c *GraphQLClient) SetAuthToken(token string) {
	c.Token = token
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message   string                 `json:"message"`
	Locations []GraphQLErrorLocation `json:"locations"`
	Path      []interface{}          `json:"path"`
}

// GraphQLErrorLocation represents error location
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Query executes a GraphQL query
func (c *GraphQLClient) Query(t *testing.T, query string, variables map[string]interface{}) *GraphQLResponse {
	requestBody := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		requestBody["variables"] = variables
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", c.BaseURL+"/graphql", bytes.NewBuffer(jsonBody))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	
	// Add requestinfo headers required by the middleware
	req.Header.Set("x-user-id", "user_integration_test")
	req.Header.Set("x-token-purpose", "test")
	req.Header.Set("x-raw-token", "aW50ZWdyYXRpb24tdGVzdC10b2tlbg==") // base64 encoded "integration-test-token"

	resp, err := c.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var graphQLResp GraphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&graphQLResp)
	require.NoError(t, err)

	return &graphQLResp
}

// UploadFile uploads a file using GraphQL multipart request
func (c *GraphQLClient) UploadFile(t *testing.T, query string, variables map[string]interface{}, fileName string, fileContent string) *GraphQLResponse {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add operations
	operations := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	operationsJSON, err := json.Marshal(operations)
	require.NoError(t, err)

	err = writer.WriteField("operations", string(operationsJSON))
	require.NoError(t, err)

	// Add map for file uploads
	mapData := map[string][]string{
		"0": {"variables.image"},
	}
	mapJSON, err := json.Marshal(mapData)
	require.NoError(t, err)

	err = writer.WriteField("map", string(mapJSON))
	require.NoError(t, err)

	// Add file
	part, err := writer.CreateFormFile("0", fileName)
	require.NoError(t, err)

	_, err = io.Copy(part, strings.NewReader(fileContent))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest("POST", c.BaseURL+"/graphql", &body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	
	// Add requestinfo headers required by the middleware
	req.Header.Set("x-user-id", "user_integration_test")
	req.Header.Set("x-token-purpose", "test")
	req.Header.Set("x-raw-token", "aW50ZWdyYXRpb24tdGVzdC10b2tlbg==") // base64 encoded "integration-test-token"

	resp, err := c.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var graphQLResp GraphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&graphQLResp)
	require.NoError(t, err)

	return &graphQLResp
}

// User represents a user in GraphQL responses
type User struct {
	ID              string  `json:"id"`
	Firstname       string  `json:"firstname"`
	Lastname        string  `json:"lastname"`
	Username        string  `json:"username"`
	Language        string  `json:"language"`
	Email           *string `json:"email"`
	ProfileImageURL *string `json:"profileImageUrl"`
}

// CreateTestUser creates a test user for integration tests
func (c *GraphQLClient) CreateTestUser(t *testing.T, userID string) *User {
	query := `
		mutation CreateUser($input: CreateUserInput!) {
			CreatUser(input: $input) {
				id
				firstname
				lastname
				username
				language
				email
				profileImageUrl
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"id":        userID,
			"firstname": "Test",
			"lastname":  "User",
			"username":  "testuser_" + userID,
			"language":  "EN",
			"email":     "test@example.com",
		},
	}

	resp := c.Query(t, query, variables)

	if len(resp.Errors) > 0 {
		t.Logf("GraphQL errors: %+v", resp.Errors)
		// User might already exist, try to get user details instead
		return c.GetUserDetails(t)
	}

	var result struct {
		CreatUser User `json:"CreatUser"`
	}
	err := json.Unmarshal(resp.Data, &result)
	require.NoError(t, err)

	return &result.CreatUser
}

// GetUserDetails gets the current user's details
func (c *GraphQLClient) GetUserDetails(t *testing.T) *User {
	query := `
		query {
			UserDetails {
				id
				firstname
				lastname
				username
				language
				email
				profileImageUrl
			}
		}
	`

	resp := c.Query(t, query, nil)

	if len(resp.Errors) > 0 {
		t.Fatalf("Failed to get user details: %+v", resp.Errors)
	}

	var result struct {
		UserDetails User `json:"UserDetails"`
	}
	err := json.Unmarshal(resp.Data, &result)
	require.NoError(t, err)

	return &result.UserDetails
}

// UploadProfileImage uploads a profile image
func (c *GraphQLClient) UploadProfileImage(t *testing.T, fileName string, fileContent string) *User {
	query := `
		mutation UploadProfileImage($image: Upload!) {
			UploadProfileImage(image: $image) {
				id
				firstname
				lastname
				username
				language
				email
				profileImageUrl
			}
		}
	`

	variables := map[string]interface{}{
		"image": nil, // This will be replaced by the file upload
	}

	resp := c.UploadFile(t, query, variables, fileName, fileContent)

	if len(resp.Errors) > 0 {
		t.Fatalf("Failed to upload profile image: %+v", resp.Errors)
	}

	var result struct {
		UploadProfileImage User `json:"UploadProfileImage"`
	}
	err := json.Unmarshal(resp.Data, &result)
	require.NoError(t, err)

	return &result.UploadProfileImage
}