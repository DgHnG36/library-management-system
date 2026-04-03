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

func TestBookInventory_EndToEnd(t *testing.T) {
	var managerToken string
	var createdBookID string

	t.Run("Step 1: List Books (Public - empty or existing)", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Books      []interface{} `json:"books"`
			TotalCount int32         `json:"total_count"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		t.Logf("total books: %d", result.TotalCount)
	})

	t.Run("Step 2: Get Non-existent Book (Public - 404)", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/books/non-existent-id", gatewayURL()),
			nil,
		)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Step 3: Login as Manager", func(t *testing.T) {
		managerToken = managerLogin(t)
		require.NotEmpty(t, managerToken)
		t.Log("manager logged in")
	})

	t.Run("Step 4: Create Books (Management)", func(t *testing.T) {
		if managerToken == "" {
			t.Skip("skipping: no manager token from previous step")
		}

		isbnSuffix := time.Now().UnixNano() % 1_000_000_000
		payload := map[string]interface{}{
			"books_payload": []map[string]interface{}{
				{
					"title":       "The Go Programming Language",
					"author":      "Alan A. A. Donovan",
					"isbn":        fmt.Sprintf("978-TEST-%d-1", isbnSuffix),
					"category":    "Programming",
					"description": "A comprehensive guide to Go",
					"quantity":    10,
				},
				{
					"title":    "Clean Code",
					"author":   "Robert C. Martin",
					"isbn":     fmt.Sprintf("978-TEST-%d-2", isbnSuffix),
					"category": "Software Engineering",
					"quantity": 5,
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

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			CreatedBooks []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"created_books"`
			SuccessCount int32 `json:"success_count"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		require.Greater(t, int(result.SuccessCount), 0, "no books created")
		require.NotEmpty(t, result.CreatedBooks)

		createdBookID = result.CreatedBooks[0].ID
		t.Logf("created %d books, first ID: %s", result.SuccessCount, createdBookID)
	})

	t.Run("Step 5: List Books (Public - after creation)", func(t *testing.T) {
		if createdBookID == "" {
			t.Skip("skipping: no books were created in previous step")
		}

		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books", gatewayURL()))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Books      []interface{} `json:"books"`
			TotalCount int32         `json:"total_count"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Greater(t, int(result.TotalCount), 0, "expected at least 1 book")
		t.Logf("total books after creation: %d", result.TotalCount)
	})

	t.Run("Step 6: Get Book By ID (Public)", func(t *testing.T) {
		if createdBookID == "" {
			t.Skip("skipping: no book ID available")
		}

		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books/%s", gatewayURL(), createdBookID))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Book struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"book"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, createdBookID, result.Book.ID)
		t.Logf("retrieved book: %s", result.Book.Title)
	})

	t.Run("Step 7: Update Book (Management)", func(t *testing.T) {
		if managerToken == "" || createdBookID == "" {
			t.Skip("skipping: no manager token or book ID available")
		}

		payload := map[string]string{
			"description": "Updated: A comprehensive guide to the Go programming language",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/v1/management/books/%s", gatewayURL(), createdBookID),
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Step 7: expected 200, got %d: %s", resp.StatusCode, string(body))
		}
		t.Log("book updated successfully")
	})

	t.Run("Step 8: Update Book Quantity (Management)", func(t *testing.T) {
		if managerToken == "" || createdBookID == "" {
			t.Skip("skipping: no manager token or book ID available")
		}

		payload := map[string]interface{}{
			"change_amount": 5,
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPatch,
			fmt.Sprintf("%s/api/v1/management/books/%s/quantity", gatewayURL(), createdBookID),
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Step 8: expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			NewAvailableQuantity int32 `json:"new_available_quantity"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		t.Logf("new available quantity: %d", result.NewAvailableQuantity)
	})

	t.Run("Step 9: Check Book Availability (Management)", func(t *testing.T) {
		if managerToken == "" || createdBookID == "" {
			t.Skip("skipping: no manager token or book ID available")
		}

		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/management/books/%s/availability", gatewayURL(), createdBookID),
			nil,
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Step 9: expected 200, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			IsAvailable       bool  `json:"is_available"`
			AvailableQuantity int32 `json:"available_quantity"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.IsAvailable)
		t.Logf("available: %v, quantity: %d", result.IsAvailable, result.AvailableQuantity)
	})

	t.Run("Step 10: Delete Book (Management)", func(t *testing.T) {
		if managerToken == "" || createdBookID == "" {
			t.Skip("skipping: no manager token or book ID available")
		}

		req, err := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("%s/api/v1/management/books/%s", gatewayURL(), createdBookID),
			nil,
		)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+managerToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Step 10: expected 200, got %d: %s", resp.StatusCode, string(body))
		}
		t.Logf("book %s deleted", createdBookID)
	})

	t.Run("Step 11: Verify Deleted Book Returns 404 (Public)", func(t *testing.T) {
		if createdBookID == "" {
			t.Skip("skipping: no book ID available")
		}

		resp, err := http.Get(fmt.Sprintf("%s/api/v1/books/%s", gatewayURL(), createdBookID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		t.Log("deleted book correctly returns 404")
	})

	t.Run("Step 12: Management Route Rejected Without Token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/management/books/some-id/availability", gatewayURL()),
			nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
