package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderFlow_EndToEnd(t *testing.T) {
	var (
		userToken    string
		managerToken string
		bookID       string
		orderID      string
		cancelID     string
	)

	t.Run("Step 1: Register test user and login", func(t *testing.T) {
		token, _ := registerAndLogin(t, "order-flow")
		userToken = token
		require.NotEmpty(t, userToken)
	})

	t.Run("Step 2: Manager login and create test book", func(t *testing.T) {
		managerToken = managerLogin(t)
		isbn := fmt.Sprintf("978-OF-%d", time.Now().UnixNano()%100_000_000)
		bookID = createTestBook(t, managerToken, "Order Flow Test Book", isbn, 10)
		require.NotEmpty(t, bookID)
	})

	t.Run("Step 3: User creates first order (to be approved)", func(t *testing.T) {
		if userToken == "" || bookID == "" {
			t.Skip("dependencies not met")
		}
		payload := map[string]interface{}{
			"book_ids":    []string{bookID},
			"borrow_days": 7,
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "create order failed: %s", string(raw))

		var result struct {
			Order struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"order"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		require.NotEmpty(t, result.Order.ID)
		assert.Equal(t, "PENDING", result.Order.Status)

		orderID = result.Order.ID
		t.Logf("created order: %s (status=%s)", orderID, result.Order.Status)
	})

	t.Run("Step 4: User creates second order (to be cancelled)", func(t *testing.T) {
		if userToken == "" || bookID == "" {
			t.Skip("dependencies not met")
		}
		// Create a second book so we have something to order independently
		isbn2 := fmt.Sprintf("978-OF2-%d", time.Now().UnixNano()%100_000_000)
		book2ID := createTestBook(t, managerToken, "Order Flow Cancel Book", isbn2, 5)

		payload := map[string]interface{}{
			"book_ids":    []string{book2ID},
			"borrow_days": 3,
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "create cancel-order failed: %s", string(raw))

		var result struct {
			Order struct {
				ID string `json:"id"`
			} `json:"order"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		cancelID = result.Order.ID
		t.Logf("created cancel-order: %s", cancelID)
	})

	t.Run("Step 5: User lists their own orders", func(t *testing.T) {
		if userToken == "" {
			t.Skip("dependencies not met")
		}
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			nil,
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "list orders failed: %s", string(raw))

		var result struct {
			Orders     []interface{} `json:"orders"`
			TotalCount int32         `json:"total_count"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		assert.GreaterOrEqual(t, int(result.TotalCount), 1)
		t.Logf("user has %d orders", result.TotalCount)
	})

	t.Run("Step 6: User gets order by ID", func(t *testing.T) {
		if userToken == "" || orderID == "" {
			t.Skip("dependencies not met")
		}
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/orders/%s", gatewayURL(), orderID),
			nil,
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "get order failed: %s", string(raw))

		var result struct {
			Order struct {
				ID string `json:"id"`
			} `json:"order"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		assert.Equal(t, orderID, result.Order.ID)
	})

	t.Run("Step 7: User cancels second order", func(t *testing.T) {
		if userToken == "" || cancelID == "" {
			t.Skip("dependencies not met")
		}
		payload := map[string]string{"cancel_reason": "changed my mind"}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders/%s/cancel", gatewayURL(), cancelID),
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "cancel order failed: %s", string(raw))

		var result struct {
			Order struct {
				Status string `json:"status"`
			} `json:"order"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		assert.Equal(t, "CANCELED", result.Order.Status)
		t.Logf("order %s cancelled", cancelID)
	})

	t.Run("Step 8: Manager lists all orders", func(t *testing.T) {
		if managerToken == "" {
			t.Skip("dependencies not met")
		}
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/management/orders", gatewayURL()),
			nil,
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "list all orders failed: %s", string(raw))

		var result struct {
			TotalCount int32 `json:"total_count"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		assert.GreaterOrEqual(t, int(result.TotalCount), 1)
		t.Logf("total orders in system: %d", result.TotalCount)
	})

	t.Run("Step 9: Manager approves first order", func(t *testing.T) {
		if managerToken == "" || orderID == "" {
			t.Skip("dependencies not met")
		}
		payload := map[string]string{"new_status": "APPROVED"}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPatch,
			fmt.Sprintf("%s/api/v1/management/orders/%s/status", gatewayURL(), orderID),
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "approve order failed: %s", string(raw))

		var result struct {
			Order struct {
				Status string `json:"status"`
			} `json:"order"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		assert.Equal(t, "APPROVED", result.Order.Status)
		t.Logf("order %s approved", orderID)
	})

	t.Run("Step 10: Manager updates order to BORROWED", func(t *testing.T) {
		if managerToken == "" || orderID == "" {
			t.Skip("dependencies not met")
		}
		payload := map[string]string{"new_status": "BORROWED", "note": "picked up at front desk"}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPatch,
			fmt.Sprintf("%s/api/v1/management/orders/%s/status", gatewayURL(), orderID),
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "borrow order failed: %s", string(raw))

		var result struct {
			Order struct {
				Status string `json:"status"`
			} `json:"order"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		assert.Equal(t, "BORROWED", result.Order.Status)
		t.Logf("order %s now BORROWED", orderID)
	})
}
