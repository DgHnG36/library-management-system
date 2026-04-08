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

/*
 *TestNotificationFlow verifies that order lifecycle events do not crash the
 *notification service.  The notification service polls AWS SQS for messages
 *and (in production) sends emails via AWS SES.  Since we cannot verify email
 *delivery in CI, this test confirms that:
 *  - Creating an order publishes an ORDER_CREATED event without error.
 *  - The gateway remains healthy after the event is published.
 *  - Cancelling an order publishes an ORDER_CANCELED event without error.
 *  - The gateway remains healthy afterwards.
 */
func TestNotificationFlow(t *testing.T) {
	var userToken, bookID, orderID string

	t.Run("Step 1: Gateway health check before test", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/healthy", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		assert.Equal(t, http.StatusOK, resp.StatusCode, "gateway should be healthy before test")
	})

	t.Run("Step 2: Setup - register user, manager login, create book", func(t *testing.T) {
		userToken, _ = registerAndLogin(t, "notif-user")
		managerToken := managerLogin(t)
		bookID = createTestBook(t, managerToken, "Notification Test Book", "ISBN-NOTIF-001", 5)
		require.NotEmpty(t, userToken)
		require.NotEmpty(t, bookID)
	})

	t.Run("Step 3: User creates an order (triggers ORDER_CREATED event)", func(t *testing.T) {
		if userToken == "" || bookID == "" {
			t.Skip("prerequisites from step 2 not met")
		}
		payload := map[string]interface{}{"book_ids": []string{bookID}, "borrow_days": 7}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders", gatewayURL()),
			bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "create order: %s", string(raw))

		var result struct {
			Order struct {
				ID string `json:"id"`
			} `json:"order"`
		}
		require.NoError(t, json.Unmarshal(raw, &result))
		orderID = result.Order.ID
		require.NotEmpty(t, orderID, "order ID must be returned")
		t.Logf("created order: %s", orderID)
	})

	t.Run("Step 4: Wait briefly for notification service to process ORDER_CREATED", func(t *testing.T) {
		if orderID == "" {
			t.Skip("no order from step 3")
		}
		// A short sleep gives the SQS consumer time to poll and acknowledge the message.
		time.Sleep(2 * time.Second)
	})

	t.Run("Step 5: Gateway remains healthy after ORDER_CREATED event", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/healthy", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		assert.Equal(t, http.StatusOK, resp.StatusCode, "gateway should stay healthy after order creation")
	})

	t.Run("Step 6: User cancels the order (triggers ORDER_CANCELED event)", func(t *testing.T) {
		if userToken == "" || orderID == "" {
			t.Skip("prerequisites from step 3 not met")
		}
		req, err := http.NewRequest(http.MethodPost,
			fmt.Sprintf("%s/api/v1/orders/%s/cancel", gatewayURL(), orderID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+userToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "cancel order: %s", string(raw))
	})

	t.Run("Step 7: Wait briefly for notification service to process ORDER_CANCELED", func(t *testing.T) {
		time.Sleep(1 * time.Second)
	})

	t.Run("Step 8: Gateway remains healthy after ORDER_CANCELED event", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/healthy", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		assert.Equal(t, http.StatusOK, resp.StatusCode, "gateway should stay healthy after order cancellation")
	})
}
