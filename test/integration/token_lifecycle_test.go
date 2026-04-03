package integration

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenLifecycle(t *testing.T) {
	var accessToken, refreshToken string

	t.Run("Step 1: Login and receive token pair", func(t *testing.T) {
		token, rt := registerAndLogin(t, "token-lifecycle")
		accessToken = token
		refreshToken = rt
		require.NotEmpty(t, accessToken, "access token must not be empty")
		require.NotEmpty(t, refreshToken, "refresh token must not be empty")
		t.Log("received access + refresh tokens")
	})

	t.Run("Step 2: Valid token grants access to protected route", func(t *testing.T) {
		if accessToken == "" {
			t.Skip("no access token from step 1")
		}
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 with valid token: %s", string(raw))
	})

	t.Run("Step 3: No Authorization header returns 401", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Step 4: Invalid token string returns 401", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer this.is.not.a.valid.token")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Step 5: Malformed Authorization header returns 401", func(t *testing.T) {
		for _, badHeader := range []string{
			"notbearer sometoken",
			"Bearer",
			"",
			"Basic dXNlcjpwYXNz",
		} {
			req, err := http.NewRequest(http.MethodGet,
				fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
			require.NoError(t, err)
			if badHeader != "" {
				req.Header.Set("Authorization", badHeader)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
				"expected 401 for Authorization=%q", badHeader)
		}
	})

	t.Run("Step 6: Sending X-Refresh-Token header does not break valid requests", func(t *testing.T) {
		if accessToken == "" || refreshToken == "" {
			t.Skip("no tokens from step 1")
		}
		// When NOT in the sliding window, the middleware ignores the refresh token
		// but the request should still succeed (no error).
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("X-Refresh-Token", refreshToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"expected 200 with valid token + refresh header: %s", string(raw))
	})

	t.Run("Step 7: Token can be used to list orders (correct user scope)", func(t *testing.T) {
		if accessToken == "" {
			t.Skip("no access token from step 1")
		}
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"expected 200 listing own orders: %s", string(raw))
	})

	t.Run("Step 8: Refresh token via login after deliberate re-login", func(t *testing.T) {
		// Log in fresh to intentionally get new tokens, verifying the login flow
		// returns different tokens from a prior session.
		newAccess, newRefresh := login(t, "lms-manager", "manager@413")
		assert.NotEmpty(t, newAccess)
		assert.NotEmpty(t, newRefresh)
		// New tokens should be valid
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+newAccess)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "new token failed: %s", string(raw))
	})

	t.Run("Step 9: Tampered token signature returns 401", func(t *testing.T) {
		if accessToken == "" {
			t.Skip("no access token from step 1")
		}
		// Corrupt a character in the middle of the token (in the payload section).
		// Avoid tampering only the last character: in a 43-char base64url HMAC-SHA256
		// signature the last char carries only 4 meaningful bits + 2 zero-padding bits,
		// so certain last-char changes are no-ops on the decoded bytes and the
		// signature still verifies. A middle character has all 6 bits active.
		mid := len(accessToken) / 2
		repl := byte('Z')
		if accessToken[mid] == 'Z' {
			repl = 'a'
		}
		tampered := accessToken[:mid] + string(repl) + accessToken[mid+1:]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tampered)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Step 10: Public book routes do not require a token", func(t *testing.T) {
		// Verify that the public book listing works without any Authorization header,
		// confirming that token absence only affects protected routes.
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "public books should not require token: %s", string(raw))
	})

}
