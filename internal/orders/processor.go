package orders

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"

	"github.com/ponty96/my-proto-schemas/output/schemas"
	"github.com/ponty96/simple-web-app/internal/db"
	"github.com/ponty96/simple-web-app/internal/rabbitmq"
)

type Processor interface {
	NewOrder(context.Context, *Order) error
}

type processor struct {
	db        *pgx.Conn
	queries   *db.Queries
	publisher rabbitmq.MQ
}

func NewProcessor(d *pgx.Conn, r rabbitmq.MQ) Processor {
	client := db.New(d)
	return &processor{
		db:        d,
		queries:   client,
		publisher: r,
	}
}

// Represents the main order structure
type Order struct {
	OrderID         *string     `json:"order_id"`
	UserID          *string     `json:"user_id"`
	Items           []OrderItem `json:"items"`
	ShippingAddress Address     `json:"shipping_address"`
	BillingAddress  Address     `json:"billing_address"`
	TotalAmount     *float64    `json:"total_amount"`
	Status          *string     `json:"status"`
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
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

func (c *processor) NewOrder(ctx context.Context, order *Order) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

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
		OrderId: *order.OrderID,
		UserId:  *order.UserID,
		Items:   items,
		// Status:  *order.Status,
		TotalAmount: *order.TotalAmount,
		ShippingAddress: &schemas.Address{
			City:  order.ShippingAddress.City,
			State: order.ShippingAddress.State,
		},
		BillingAddress: &schemas.Address{
			City:  order.BillingAddress.City,
			State: order.BillingAddress.State,
		},
	}
	// convert
	if err := c.publisher.Publish(ctx, &o); err != nil {
		// we need to log an error here.
		logrus.Errorf("Failed to publish: %v", err)
		return err
	}
	return nil
}
