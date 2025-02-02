package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/ponty96/my-proto-schemas/output/schemas"
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
		log.Error(err)
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
		log.Error(err)
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

	var items []*schemas.OrderItem

	for _, i := range order.Items {
		it := i
		items = append(items, &schemas.OrderItem{
			Price:      it.Price,
			ProductId:  it.ProductID,
			Quantity:   it.Quantity,
			TotalPrice: it.TotalPrice,
		})
	}

	o := schemas.Order{
		OrderId:     *order.OrderID,
		UserId:      *order.UserID,
		Items:       items,
		OrderStatus: *order.Status,
		TotalAmount: *order.TotalAmount,
		ShippingAddress: &schemas.Address{
			City:   order.ShippingAddress.City,
			State:  order.ShippingAddress.State,
			Street: order.ShippingAddress.Line1,
		},
		BillingAddress: &schemas.Address{
			City:   order.BillingAddress.City,
			State:  order.BillingAddress.State,
			Street: order.BillingAddress.Line1,
		},
	}

	// if o.Validate() != nil {

	// }

	if err = s.Config.MQ.Publish(ctx, &o); err != nil {
		log.Errorf("failed to publish %v", err)
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
