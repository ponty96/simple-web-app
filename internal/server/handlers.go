package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/pkg/errors"
)

type Timestamp struct {
	Seconds int64 `json:"seconds"`
	Nanos   int32 `json:"nanos"`
}

// Represents the main order structure
type Order struct {
	OrderID         *string      `json:"order_id"`
	UserID          *string      `json:"user_id"`
	Items           []OrderItem  `json:"items"`
	ShippingAddress Address      `json:"shipping_address"`
	BillingAddress  Address      `json:"billing_address"`
	TotalAmount     *float64     `json:"total_amount"`
	Status          *OrderStatus `json:"status"`
	CreatedAt       Timestamp    `json:"created_at"`
	UpdatedAt       Timestamp    `json:"updated_at"`
}

// Represents a single item within an order
type OrderItem struct {
	ProductID  string  `json:"product_id"`
	Quantity   int32   `json:"quantity"`
	Price      float64 `json:"price"`
	TotalPrice float64 `json:"total_price"`
}

// Represents an address used for shipping or billing
type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// Status of the order
type OrderStatus string

const (
	OrderStatusUnspecified OrderStatus = "ORDER_STATUS_UNSPECIFIED"
	OrderStatusPending     OrderStatus = "PENDING"
	OrderStatusShipped     OrderStatus = "SHIPPED"
	OrderStatusDelivered   OrderStatus = "DELIVERED"
	OrderStatusCancelled   OrderStatus = "CANCELLED"
)

func (s *server) orderWebhookHandler(w http.ResponseWriter, r *http.Request) {
	// read the request payload which should be a json
	// validate required fields
	body, err := io.ReadAll(r.Body)
	if err != nil {
		err := errors.Wrap(err, "failed to read request")
		log.Print(err)
		httpWriteJSON(w, Response{
			Message: "",
			Code:    http.StatusInternalServerError,
		})
	}

	var order Order

	err = json.Unmarshal(body, &order)

	if err != nil {
		err := errors.Wrap(err, "failed to parse json")
		log.Print(err)
		httpWriteJSON(w, Response{
			Message: "invalid json",
			Code:    http.StatusUnprocessableEntity,
		})
	}

	v := make(map[string]string)
	if order.OrderID == nil {
		v["order_id"] = "is required"
	}
	if order.UserID == nil {
		v["user_id"] = "is required"
	}

	if order.TotalAmount == nil {
		v["total_amount"] = "is required"
	}

	if order.Status == nil {
		v["status"] = "is required"
	}

	if len(v) > 0 {
		httpWriteJSON(w, Response{
			Message: "validation failed",
			Code:    http.StatusUnprocessableEntity,
			Errs:    v,
		})
	}

	httpWriteJSON(w, Response{
		Message: "Order Created",
		Code:    http.StatusCreated,
	})
}
