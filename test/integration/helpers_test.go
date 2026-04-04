package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func gatewayURL() string {
	if url := os.Getenv("GATEWAY_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func login(t *testing.T, identifier, password string) (accessToken, refreshToken string) {
	t.Helper()
	payload := map[string]string{"identifier": identifier, "password": password}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/auth/login", gatewayURL()),
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode, "login failed for %s: %s", identifier, string(raw))

	var result struct {
		TokenPair struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"token_pair"`
	}
	require.NoError(t, json.Unmarshal(raw, &result))
	require.NotEmpty(t, result.TokenPair.AccessToken)
	return result.TokenPair.AccessToken, result.TokenPair.RefreshToken
}

func managerLogin(t *testing.T) string {
	t.Helper()
	username := "lms-manager"
	password := "manager@413"
	token, _ := login(t, username, password)
	return token
}

func adminLogin(t *testing.T) string {
	t.Helper()
	username := "lms-admin"
	password := "@dm1n79"
	token, _ := login(t, username, password)
	return token
}

func registerAndLogin(t *testing.T, prefix string) (accessToken, userID string) {
	t.Helper()
	suffix := time.Now().UnixNano() % 1_000_000_000
	username := fmt.Sprintf("%s-%d", prefix, suffix)
	email := fmt.Sprintf("%s-%d@test.local", prefix, suffix)
	password := "testpass123"

	regPayload := map[string]string{
		"username":     username,
		"password":     password,
		"email":        email,
		"phone_number": "0900000000",
	}
	regBody, err := json.Marshal(regPayload)
	require.NoError(t, err)

	regResp, err := http.Post(
		fmt.Sprintf("%s/api/v1/auth/register", gatewayURL()),
		"application/json",
		bytes.NewReader(regBody),
	)
	require.NoError(t, err)
	defer regResp.Body.Close()
	raw, _ := io.ReadAll(regResp.Body)
	require.Equal(t, http.StatusCreated, regResp.StatusCode, "register failed: %s", string(raw))

	var regResult struct {
		UserID string `json:"user_id"`
	}
	require.NoError(t, json.Unmarshal(raw, &regResult))
	userID = regResult.UserID

	accessToken, _ = login(t, username, password)
	return accessToken, userID
}

func createTestBook(t *testing.T, managerToken, title, isbn string, quantity int32) string {
	t.Helper()
	payload := map[string]interface{}{
		"books_payload": []map[string]interface{}{
			{
				"title":    title,
				"author":   "Test Author",
				"isbn":     isbn,
				"category": "Test",
				"quantity": quantity,
			},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/api/v1/management/books", gatewayURL()),
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+managerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "createTestBook failed: %s", string(raw))

	var result struct {
		CreatedBooks []struct {
			ID string `json:"id"`
		} `json:"created_books"`
	}
	require.NoError(t, json.Unmarshal(raw, &result))
	require.NotEmpty(t, result.CreatedBooks)
	return result.CreatedBooks[0].ID
}
