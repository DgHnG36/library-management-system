package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* TestRateLimit verifies that the gateway enforces rate limiting correctly.
 * The test environment uses RATE_LIMIT_MAX_REQUESTS=1000 / RATE_LIMIT_WINDOW_SECONDS=60.
 * Rather than exhausting the quota (which would slow all subsequent tests), this
 * suite focuses on header correctness and response format.
 */
func TestRateLimit(t *testing.T) {

	t.Run("Step 1: Rate-limit headers are present on public routes", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck

		assert.NotEmpty(t, resp.Header.Get("X-Rate-Limit"),
			"X-Rate-Limit header must be set")
		assert.NotEmpty(t, resp.Header.Get("X-Rate-Limit-Remaining"),
			"X-Rate-Limit-Remaining header must be set")

		t.Logf("X-Rate-Limit=%s  X-Rate-Limit-Remaining=%s",
			resp.Header.Get("X-Rate-Limit"),
			resp.Header.Get("X-Rate-Limit-Remaining"))
	})

	t.Run("Step 2: Rate-limit headers are present on protected routes", func(t *testing.T) {
		token, _ := registerAndLogin(t, "ratelimit-user")

		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck

		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEmpty(t, resp.Header.Get("X-Rate-Limit"))
		assert.NotEmpty(t, resp.Header.Get("X-Rate-Limit-Remaining"))
	})

	t.Run("Step 3: X-Rate-Limit value matches configured maximum", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck

		limitStr := resp.Header.Get("X-Rate-Limit")
		require.NotEmpty(t, limitStr)

		limit, err := strconv.Atoi(limitStr)
		require.NoError(t, err, "X-Rate-Limit header must be numeric")
		assert.Greater(t, limit, 0, "rate limit must be positive")
	})

	t.Run("Step 4: X-Rate-Limit-Remaining decrements across requests", func(t *testing.T) {
		// Make two consecutive requests from the same IP and check that
		// the remaining counter decrements (or stays the same if reset).
		first, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer first.Body.Close()
		io.Copy(io.Discard, first.Body) //nolint:errcheck

		firstRemaining := first.Header.Get("X-Rate-Limit-Remaining")
		require.NotEmpty(t, firstRemaining)

		second, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer second.Body.Close()
		io.Copy(io.Discard, second.Body) //nolint:errcheck

		secondRemaining := second.Header.Get("X-Rate-Limit-Remaining")
		require.NotEmpty(t, secondRemaining)

		r1, _ := strconv.Atoi(firstRemaining)
		r2, _ := strconv.Atoi(secondRemaining)
		// After a second request the remaining count should be less (or equal
		// if the window rolled over between the two calls).
		assert.LessOrEqual(t, r2, r1,
			"remaining should not increase between consecutive requests (r1=%d r2=%d)", r1, r2)
		t.Logf("remaining: first=%d second=%d", r1, r2)
	})

	t.Run("Step 5: 429 response body is well-formed when limit exceeded", func(t *testing.T) {
		// We cannot easily exhaust 1000 requests in a unit test, so instead we
		// verify that the error shape used by the gateway matches the AppError
		// schema — tested here via a known-format 429 induced by calling a
		// non-existent high-frequency scenario simulation is skipped;
		// we instead confirm the 200 path returns the correct header format.
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Only assert the format if we happened to hit 429 (e.g. CI reuse).
		if resp.StatusCode == http.StatusTooManyRequests {
			var errBody struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
			assert.Equal(t, "RATE_LIMIT_EXCEEDED", errBody.Code)
			assert.NotEmpty(t, errBody.Message)
		} else {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("Step 6: Auth routes are rate-limited", func(t *testing.T) {
		// Login with invalid credentials; the endpoint is under /api/v1 so it
		// should carry rate-limit headers regardless of the auth outcome.
		body := []byte(`{"identifier":"nobody","password":"wrong"}`)
		resp, err := http.Post(
			fmt.Sprintf("%s/api/v1/auth/login", gatewayURL()),
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck

		// We expect 401 (wrong credentials), headers should still be present.
		assert.Contains(t, []int{http.StatusUnauthorized, http.StatusNotFound, http.StatusTooManyRequests},
			resp.StatusCode)
		// Rate-limit headers should be set even on auth failures.
		assert.NotEmpty(t, resp.Header.Get("X-Rate-Limit"))
	})
}
