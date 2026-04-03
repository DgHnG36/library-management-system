package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow_EndToEnd(t *testing.T) {
	testUser := struct {
		Username    string
		Password    string
		Email       string
		PhoneNumber string
	}{
		Username:    "user-test-auth-e2e",
		Password:    "passuserauth",
		Email:       "user-test-auth-e2e@example.com",
		PhoneNumber: "0967125598",
	}

	var accessToken string

	t.Run("Step 1: Register New User", func(t *testing.T) {
		payload := map[string]string{
			"username":     testUser.Username,
			"password":     testUser.Password,
			"email":        testUser.Email,
			"phone_number": testUser.PhoneNumber,
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err, "failed to marshal register payload: %v", err)

		resp, err := http.Post(
			fmt.Sprintf("%s/api/v1/auth/register", gatewayURL()),
			"application/json",
			bytes.NewReader(body),
		)
		assert.NoError(t, err, "register request failed: %v", err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode, "expected %d, got %d", http.StatusCreated, resp.StatusCode)

		var result struct {
			UserID string `json:"user_id"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "failed to decode register response: %v", err)

		assert.NotEmpty(t, result.UserID, "register response missing user_id")
		t.Logf("registered user_id: %s", result.UserID)
	})

	t.Run("Step 2: Login", func(t *testing.T) {
		payload := map[string]string{
			"identifier": testUser.Username,
			"password":   testUser.Password,
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err, "failed to marshal register payload: %v", err)

		resp, err := http.Post(
			fmt.Sprintf("%s/api/v1/auth/login", gatewayURL()),
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err, "login request failed: %v", err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected %d, got %d", http.StatusOK, resp.StatusCode)

		var result struct {
			TokenPair struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"token_pair"`
			User struct {
				ID       string `json:"id"`
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"user"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "failed to decode login response: %v", err)
		require.NotEmpty(t, result.TokenPair.AccessToken, "login response missing access_token")

		accessToken = result.TokenPair.AccessToken
		t.Logf("logged in as user_id: %s", result.User.ID)
	})

	t.Run("Step 3: Get Profile", func(t *testing.T) {
		if accessToken == "" {
			t.Skip("skipping: no access token from previous step")
		}

		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()),
			nil,
		)
		require.NoError(t, err, "failed to create profile request: %v", err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "get profile request failed: %v", err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected %d, got %d", http.StatusOK, resp.StatusCode)

		var result struct {
			User struct {
				ID       string `json:"id"`
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"user"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "failed to decode profile response: %v", err)
		require.Equal(t, testUser.Username, result.User.Username, "expected username %q, got %q", testUser.Username, result.User.Username)
		t.Logf("profile retrieved for user: %s", result.User.Username)
	})

	t.Run("Step 4: Update Profile", func(t *testing.T) {
		if accessToken == "" {
			t.Skip("skipping: no access token from previous step")
		}

		payload := map[string]string{
			"phone_number": "0912345678",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err, "failed to marshal update payload: %v", err)

		req, err := http.NewRequest(http.MethodPatch,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()),
			bytes.NewReader(body),
		)
		require.NoError(t, err, "failed to create update profile request: %v", err)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("update profile request failed: %v", err)
		}
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected %d, got %d", http.StatusOK, resp.StatusCode)
		t.Log("profile updated successfully")
	})

	t.Run("Step 5: Access Protected Route Without Token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()),
			nil,
		)
		require.NoError(t, err, "failed to create request: %v", err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "request failed: %v", err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 Unauthorized, got %d", resp.StatusCode)
	})
}
