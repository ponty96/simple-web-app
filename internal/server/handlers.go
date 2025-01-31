package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/ponty96/simple-web-app/internal/orders"
)

func (s *server) orderWebhookHandler(w http.ResponseWriter, r *http.Request) {
	// read the request payload which should be a json
	// validate required fields
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second) // maximum timeout of the webhook provider to wait for a response
	defer cancel()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		err := errors.Wrap(err, "failed to read request")
		log.Print(err)
		httpWriteJSON(w, Response{
			Message: "",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	var order orders.Order

	err = json.Unmarshal(body, &order)

	if err != nil {
		err := errors.Wrap(err, "failed to parse json")
		log.Print(err)
		httpWriteJSON(w, Response{
			Message: "invalid json",
			Code:    http.StatusUnprocessableEntity,
		})
		return
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
		return
	}

	if err = s.Config.Processor.NewOrder(ctx, &order); err != nil {
		httpWriteJSON(w, Response{
			Message: "failed to process order",
			Code:    http.StatusInternalServerError,
		})
	} else {
		httpWriteJSON(w, Response{
			Message: "Order Created",
			Code:    http.StatusCreated,
		})
	}
}
