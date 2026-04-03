package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* TestServiceResilience verifies that the gateway handles bad input gracefully
 * and that infrastructure health endpoints are reachable.  All assertions are
 * made against well-specified error responses (no 500s for client mistakes).
 */
func TestServiceResilience(t *testing.T) {
	t.Run("Step 1: GET /healthy returns 200", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/healthy", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "/healthy: %s", string(raw))
	})

	t.Run("Step 2: GET /ready returns 200 (all dependencies reachable)", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/ready", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "/ready: %s", string(raw))
	})

	t.Run("Step 3: Order with non-existent book ID returns client error (not 500)", func(t *testing.T) {
		token, _ := registerAndLogin(t, "resilience-user")

		payload := map[string]interface{}{
			"book_ids":    []string{"00000000-0000-0000-0000-000000000000"},
			"borrow_days": 7,
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		// Must not be a server error — expect 4xx
		assert.True(t, resp.StatusCode >= 400 && resp.StatusCode < 500,
			"non-existent book should return 4xx, got %d: %s", resp.StatusCode, string(raw))
	})

	t.Run("Step 4: Invalid JSON body returns 400", func(t *testing.T) {
		token, _ := registerAndLogin(t, "resilience-badjson")

		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			bytes.NewReader([]byte(`{this is not json}`)))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"invalid JSON should return 400: %s", string(raw))
	})

	t.Run("Step 5: Missing required fields in CreateOrder returns 400", func(t *testing.T) {
		token, _ := registerAndLogin(t, "resilience-missingfields")

		// Send an empty JSON object — all required fields are absent.
		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			bytes.NewReader([]byte(`{}`)))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"empty body should return 400: %s", string(raw))
	})

	t.Run("Step 6: Empty book_ids array returns 400", func(t *testing.T) {
		token, _ := registerAndLogin(t, "resilience-emptybookids")

		payload := map[string]interface{}{"book_ids": []string{}, "borrow_days": 7}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"empty book_ids should return 400: %s", string(raw))
	})

	t.Run("Step 7: Get order with non-existent UUID returns 404", func(t *testing.T) {
		token, _ := registerAndLogin(t, "resilience-getorder")

		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/orders/00000000-0000-0000-0000-000000000000", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode,
			"non-existent order should return 404: %s", string(raw))
	})

	t.Run("Step 8: Register with duplicate username returns 409", func(t *testing.T) {
		token, _ := registerAndLogin(t, "resilience-dup")
		require.NotEmpty(t, token)

		const fixedUsername = "resilience-dup-conflict"
		regPayload := map[string]string{
			"username":     fixedUsername,
			"password":     "testpass123",
			"email":        fixedUsername + "@test.local",
			"phone_number": "0900000001",
		}
		body, err := json.Marshal(regPayload)
		require.NoError(t, err)

		firstResp, err := http.Post(
			fmt.Sprintf("%s/api/v1/auth/register", gatewayURL()),
			"application/json",
			bytes.NewReader(body))
		require.NoError(t, err)
		io.Copy(io.Discard, firstResp.Body) //nolint:errcheck
		firstResp.Body.Close()

		body2, _ := json.Marshal(regPayload)
		secondResp, err := http.Post(
			fmt.Sprintf("%s/api/v1/auth/register", gatewayURL()),
			"application/json",
			bytes.NewReader(body2))
		require.NoError(t, err)
		defer secondResp.Body.Close()
		raw, _ := io.ReadAll(secondResp.Body)
		assert.Equal(t, http.StatusConflict, secondResp.StatusCode,
			"duplicate registration should return 409: %s", string(raw))
	})
}
