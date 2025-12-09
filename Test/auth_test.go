package controllers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	// Change this to your actual server URL
	BASE_URL = "http://localhost:8080"
	API_BASE = BASE_URL + "/api/v1"
)

// HTTP Client with timeout
var client = &http.Client{
	Timeout: 10 * time.Second,
}

// Helper to generate unique email for each test run
func generateUniqueEmail(prefix string) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s_%d@example.com", prefix, rand.Intn(1000000))
}

// Helper function to make HTTP requests to real server
func makeRequest(method, url string, body interface{}, token string) (*http.Response, []byte, error) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Add token if provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// Read response body
	var responseBody []byte
	if resp.Body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		responseBody = buf.Bytes()
	}

	return resp, responseBody, nil
}

func TestRegister_Success(t *testing.T) {
	payload := map[string]interface{}{
		"email":    generateUniqueEmail("test"),
		"password": "password123",
		"name":     "Test User",
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/register", payload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "user")
	assert.Contains(t, response, "token")
	assert.NotEmpty(t, response["token"])

}

func TestRegister_InvalidEmail(t *testing.T) {
	payload := map[string]interface{}{
		"email":    "invalid-email",
		"password": "password123",
		"name":     "Test User",
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/register", payload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
}

func TestRegister_ShortPassword(t *testing.T) {
	payload := map[string]interface{}{
		"email":    generateUniqueEmail("test2"),
		"password": "123",
		"name":     "Test User",
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/register", payload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
}

func TestRegister_MissingFields(t *testing.T) {
	payload := map[string]interface{}{
		"email": generateUniqueEmail("test3"),
		// missing password and name
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/register", payload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	// Use same email for both attempts
	email := generateUniqueEmail("duplicate")

	payload := map[string]interface{}{
		"email":    email,
		"password": "password123",
		"name":     "Test User",
	}

	// First registration
	resp1, _, err := makeRequest("POST", API_BASE+"/auth/register", payload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	// Attempt duplicate registration with SAME email
	resp2, body, err := makeRequest("POST", API_BASE+"/auth/register", payload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
}

func TestLogin_Success(t *testing.T) {
	// First register a user with unique email
	email := generateUniqueEmail("login")
	registerPayload := map[string]interface{}{
		"email":    email,
		"password": "password123",
		"name":     "Login User",
	}
	makeRequest("POST", API_BASE+"/auth/register", registerPayload, "")

	// Now login with same credentials
	loginPayload := map[string]interface{}{
		"email":    email,
		"password": "password123",
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/login", loginPayload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "user")
	assert.Contains(t, response, "token")
	assert.NotEmpty(t, response["token"])
}

func TestLogin_InvalidCredentials(t *testing.T) {
	loginPayload := map[string]interface{}{
		"email":    generateUniqueEmail("nonexistent"),
		"password": "wrongpassword",
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/login", loginPayload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
}

func TestLogin_WrongPassword(t *testing.T) {
	// Register a user
	email := generateUniqueEmail("wrongpass")
	registerPayload := map[string]interface{}{
		"email":    email,
		"password": "correctpassword",
		"name":     "Test User",
	}
	makeRequest("POST", API_BASE+"/auth/register", registerPayload, "")

	// Try login with wrong password
	loginPayload := map[string]interface{}{
		"email":    email,
		"password": "wrongpassword",
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/login", loginPayload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "error")
}

func TestLogin_InvalidEmailFormat(t *testing.T) {
	loginPayload := map[string]interface{}{
		"email":    "note-an-email",
		"password": "password123",
	}

	resp, body, err := makeRequest("POST", API_BASE+"/auth/login", loginPayload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestLogin_MissingFields(t *testing.T) {
	loginPayload := map[string]interface{}{
		"email": generateUniqueEmail("test"),
		//password missing
	}
	resp, body, err := makeRequest("POST", API_BASE+"/auth/login", loginPayload, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

// Test protected endpoint with valid token
func TestProtectedEndpoint_WithValidToken(t *testing.T) {
	// Register and get token
	email := generateUniqueEmail("protected")
	registerPayload := map[string]interface{}{
		"email":    email,
		"password": "password123",
		"name":     "Protected User",
	}
	_, body, err := makeRequest("POST", API_BASE+"/auth/register", registerPayload, "")
	assert.NoError(t, err)

	var registerRespose map[string]interface{}
	err = json.Unmarshal(body, &registerRespose)
	assert.NoError(t, err)

	if registerRespose["token"] == nil {
		t.Fatalf("Token not found in register response. Response: %+v", registerRespose)
	}
	token := registerRespose["token"].(string)

	resp, _, err := makeRequest("GET", API_BASE+"/products", nil, token)
	assert.NoError(t, err)
	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProtectedEndpoint_WithoutToken(t *testing.T) {
	resp, _, err := makeRequest("GET", API_BASE+"/products", nil, "")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProtectedEndpoint_WithInvalidToken(t *testing.T) {
	resp, _, err := makeRequest("GET", API_BASE+"/products", nil, "invalid-token-here")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// // Test example with custom headers
func TestLogin_WithCustomHeaders(t *testing.T) {
	// Register user first
	email := generateUniqueEmail("custom")
	registerPayload := map[string]interface{}{
		"email":    email,
		"password": "password123",
		"name":     "Custom User",
	}
	makeRequest("POST", API_BASE+"/auth/register", registerPayload, "")

	// Login with custom headers
	loginPayload := map[string]interface{}{
		"email":    email,
		"password": "password123",
	}

	reqBody, _ := json.Marshal(loginPayload)
	req, _ := http.NewRequest("POST", API_BASE+"/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Go-Test-Client")
	req.Header.Set("X-Request-ID", "test-123")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
