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

func TestRBAC_Access(t *testing.T) {
	var userToken, userID string
	var managerToken, adminToken string

	t.Run("Step 1: Setup - obtain credentials for all roles", func(t *testing.T) {
		userToken, userID = registerAndLogin(t, "rbac-user")
		managerToken = managerLogin(t)
		adminToken = adminLogin(t)
		require.NotEmpty(t, userToken, "user token must not be empty")
		require.NotEmpty(t, userID, "user ID must not be empty")
		require.NotEmpty(t, managerToken, "manager token must not be empty")
		require.NotEmpty(t, adminToken, "admin token must not be empty")
	})

	t.Run("Step 2: No token returns 401 on protected routes", func(t *testing.T) {
		for _, path := range []string{"/api/v1/user/profile", "/api/v1/orders"} {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", gatewayURL(), path), nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
			resp.Body.Close()
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "unauthenticated path=%s should be 401", path)
		}
	})

	t.Run("Step 3: USER token can access own profile (200)", func(t *testing.T) {
		if userToken == "" {
			t.Skip("no user token from step 1")
		}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/user/profile", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "USER /user/profile: %s", string(raw))
	})

	t.Run("Step 4: USER token is denied on management routes (403)", func(t *testing.T) {
		if userToken == "" {
			t.Skip("no user token from step 1")
		}
		for _, path := range []string{
			"/api/v1/management/users",
			"/api/v1/management/orders",
		} {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", gatewayURL(), path), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+userToken)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			raw, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.Equal(t, http.StatusForbidden, resp.StatusCode,
				"USER on management path=%s should be 403: %s", path, string(raw))
		}
	})

	t.Run("Step 5: USER token is denied on admin-only routes (403)", func(t *testing.T) {
		if userToken == "" || userID == "" {
			t.Skip("no user token/id from step 1")
		}
		req, err := http.NewRequest(http.MethodPatch,
			fmt.Sprintf("%s/api/v1/admin/users/%s/vip", gatewayURL(), userID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "USER on admin route: %s", string(raw))
	})

	t.Run("Step 6: MANAGER token can access management orders (200)", func(t *testing.T) {
		if managerToken == "" {
			t.Skip("no manager token from step 1")
		}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/management/orders", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "MANAGER on management/orders: %s", string(raw))
	})

	t.Run("Step 7: MANAGER token can access management users (200)", func(t *testing.T) {
		if managerToken == "" {
			t.Skip("no manager token from step 1")
		}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/management/users", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "MANAGER on management/users: %s", string(raw))
	})

	t.Run("Step 8: MANAGER token is denied on admin-only routes (403)", func(t *testing.T) {
		if managerToken == "" || userID == "" {
			t.Skip("no manager token or user ID from step 1")
		}
		req, err := http.NewRequest(http.MethodPatch,
			fmt.Sprintf("%s/api/v1/admin/users/%s/vip", gatewayURL(), userID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "MANAGER on admin route: %s", string(raw))
	})

	t.Run("Step 9: ADMIN token can access management routes (200)", func(t *testing.T) {
		if adminToken == "" {
			t.Skip("no admin token from step 1")
		}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/management/orders", gatewayURL()), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "ADMIN on management/orders: %s", string(raw))
	})

	t.Run("Step 10: ADMIN token can access admin-only route (200)", func(t *testing.T) {
		if adminToken == "" || userID == "" {
			t.Skip("no admin token or user ID from step 1")
		}

		vipBody, _ := json.Marshal(map[string]bool{"is_vip": true})
		req, err := http.NewRequest(http.MethodPatch,
			fmt.Sprintf("%s/api/v1/admin/users/%s/vip", gatewayURL(), userID),
			bytes.NewReader(vipBody))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 300,
			"ADMIN on /admin/users/:id/vip should succeed: status=%d body=%s", resp.StatusCode, string(raw))
	})
}
