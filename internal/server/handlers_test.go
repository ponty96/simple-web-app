package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ponty96/simple-web-app/internal/orders"
)

// ---- processor.Processor Mock for Testing --- //
type ProcessorMock struct {
	newOrder *orders.Order
}

func (c *ProcessorMock) NewOrder(ctx context.Context, order *orders.Order) error {
	c.newOrder = order
	return nil
}

// --- End of processor.Processor Mock ---- //

func Test_OrderWebhookBadRequest(t *testing.T) {
	cfg := &Config{
		Host: "localhost",
		Port: 4050,
	}

	s := NewHTTP(cfg)

	payload := `
		"order_id": {
		  id: dddd
		}
	`
	req := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	w := httptest.NewRecorder()

	s.orderWebhookHandler(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected Unprocessable entity, got %d", w.Code)
	}

	// check Response
	var r Response

	if err := json.NewDecoder(w.Body).Decode(&r); err != nil {
		t.Fatalf("Failed to decode response %v", err)
	}

	if r.Message != "invalid json" {
		t.Errorf("Expected %s, got %s", "invalid json", r.Message)
	}
}

func Test_OrderWebhookRequiredFields(t *testing.T) {
	requiredFields := []string{"order_id", "user_id", "total_amount", "status"}

	for _, field := range requiredFields {
		t.Run(field, func(t *testing.T) {
			// Create payload with missing field
			payload := map[string]interface{}{
				"order_id":     "order-123",
				"user_id":      "user-456",
				"total_amount": 39.98,
				"status":       "PENDING",
			}
			delete(payload, field)

			jsonPayload, _ := json.Marshal(payload)

			// Setup and make request
			cfg := &Config{Host: "localhost", Port: 4050}
			s := NewHTTP(cfg)
			req := httptest.NewRequest("POST", "/", bytes.NewReader(jsonPayload))
			w := httptest.NewRecorder()

			s.orderWebhookHandler(w, req)

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("Missing %s: expected status 422, got %d", field, w.Code)
			}

			// check Response
			var r Response

			if err := json.NewDecoder(w.Body).Decode(&r); err != nil {
				t.Fatalf("Failed to decode response %v", err)
			}

			if msg, ok := r.Errs[field]; !ok {
				t.Errorf("Expected validation error for %s", field)
			} else if msg != "is required" {
				t.Errorf("Expected error message 'is required', got '%s'", msg)
			}

		})
	}
}

func Test_OrderWebhookSuccessRequest(t *testing.T) {
	p := &ProcessorMock{}
	cfg := &Config{
		Host:      "localhost",
		Port:      4050,
		Processor: p,
	}

	s := NewHTTP(cfg)

	// Example JSON payload for an order
	payload := `{
        "order_id": "test-123",
        "user_id": "user-456",
        "items": [
            {
                "product_id": "p-789",
                "quantity": 2,
                "price": 19.99,
                "total_price": 39.98
            }
        ],
        "shipping_address": {
            "line1": "123 Example St",
            "city": "ExampleCity",
            "state": "CA",
            "postal_code": "12345",
            "country": "US"
        },
        "billing_address": {
            "line1": "456 Billing Ave",
            "city": "BillingCity",
            "state": "NY",
            "postal_code": "98765",
            "country": "US"
        },
        "total_amount": 39.98,
        "status": "PENDING",
        "created_at": {
            "seconds": 1680101010,
            "nanos": 0
        },
        "updated_at": {
            "seconds": 1680101010,
            "nanos": 0
        }
    }`

	req := httptest.NewRequest("POST", "/", strings.NewReader(payload))
	w := httptest.NewRecorder()

	s.orderWebhookHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected %d, got %d", http.StatusCreated, w.Code)
	}

	// check Response
	var r Response

	if err := json.NewDecoder(w.Body).Decode(&r); err != nil {
		t.Fatalf("Failed to decode response %v", err)
	}

	if r.Code != http.StatusCreated {
		t.Errorf("Expected %d, got %d", http.StatusCreated, r.Code)
	}

	if *p.newOrder.OrderID != "test-123" {
		t.Error("failed to call processor")
	}
}
