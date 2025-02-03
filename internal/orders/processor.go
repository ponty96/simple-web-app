package orders

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	log "github.com/sirupsen/logrus"

	"github.com/ponty96/my-proto-schemas/output/schemas"
	"github.com/ponty96/simple-web-app/internal/db"
)

type Processor interface {
	NewOrder(context.Context, proto.Message) error
	ListUserOrders(context.Context, string) ([]Order, error)
}

type processor struct {
	db      *pgx.Conn
	queries *db.Queries
	// publisher rabbitmq.MQ
}

func NewProcessor(d *pgx.Conn) *processor {
	client := db.New(d)
	return &processor{
		db:      d,
		queries: client,
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

func (p *processor) NewOrder(ctx context.Context, msg proto.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	o, ok := msg.(*schemas.Order)

	if !ok {
		return fmt.Errorf("unexpected message type: %T", msg)
	}

	var shippingAddressID pgtype.UUID = pgtype.UUID{Valid: false}
	var billingAddressID pgtype.UUID = pgtype.UUID{Valid: false}

	if o.ShippingAddress != nil && o.ShippingAddress.Street != "" {
		sAdd, err := p.queries.CreateAddress(ctx, &db.CreateAddressParams{
			Line1:   o.ShippingAddress.Street,
			State:   o.ShippingAddress.State,
			City:    o.ShippingAddress.City,
			Country: "GB",
		})
		if err != nil {
			return errors.Wrap(err, "failed to insert shipping address")
		}

		shippingAddressID = sAdd.ID
	}

	if o.BillingAddress != nil && o.BillingAddress.Street != "" {
		bAdd, err := p.queries.CreateAddress(ctx, &db.CreateAddressParams{
			Line1:   o.BillingAddress.Street,
			State:   o.BillingAddress.State,
			City:    o.BillingAddress.City,
			Country: "GB",
		})
		if err != nil {
			return errors.Wrap(err, "failed to insert billing address")
		}
		billingAddressID = bAdd.ID
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(o.UserId); err != nil {
		return errors.Wrap(err, "failed to parse UUID")
	}

	var totalAmount pgtype.Numeric
	if err := totalAmount.Scan(fmt.Sprintf("%.2f", o.TotalAmount)); err != nil {
		return errors.Wrap(err, "failed to convert total amount to numeric")
	}

	insertedOrder, err := p.queries.CreateOrder(ctx, &db.CreateOrderParams{
		ShippingAddressID: shippingAddressID,
		BillingAddressID:  billingAddressID,
		UserID:            userUUID,
		Status:            db.OrderStatus(o.OrderStatus),
		TotalAmount:       totalAmount,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create order")
	}

	for _, item := range o.GetItems() {
		var price pgtype.Numeric
		if err := price.Scan(fmt.Sprintf("%.2f", item.Price)); err != nil {
			return errors.Wrap(err, "failed to convert price to numeric")
		}

		var tP pgtype.Numeric
		if err := tP.Scan(fmt.Sprintf("%.2f", item.TotalPrice)); err != nil {
			return errors.Wrap(err, "failed to convert total price to numeric")
		}

		var productUUID pgtype.UUID
		if err := productUUID.Scan(item.ProductId); err != nil {
			return errors.Wrap(err, "failed to parse UUID")
		}

		_, err := p.queries.CreateOrderItem(ctx, &db.CreateOrderItemParams{
			OrderID:    insertedOrder.ID,
			Price:      price,
			Quantity:   item.Quantity,
			TotalPrice: tP,
			ProductID:  productUUID,
		})

		if err != nil {
			return errors.Wrap(err, "failed to create order item")
		}
	}

	log.Print("Successfully created an order")

	return nil
}

func (p *processor) ListUserOrders(ctx context.Context, ID string) ([]Order, error) {
	var userID pgtype.UUID

	if err := userID.Scan(ID); err != nil {
		return nil, errors.Wrap(err, "failed to parse UUID")
	}

	orders, err := p.queries.ListOrders(ctx, userID)

	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch orders")
	}

	var os []Order

	for _, o := range orders {
		shippingAddress, err := p.queries.GetAddress(ctx, o.ShippingAddressID)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to fetch shipping address for %v", o.ID))
		}

		billingAddress, err := p.queries.GetAddress(ctx, o.BillingAddressID)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to fetch billing address for %v", o.ID))
		}

		items, err := p.queries.ListOrderItems(ctx, o.ID)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to fetch orders items for %v", o.ID))
		}

		var orderItems []OrderItem

		for _, item := range items {
			itemPrice, _ := item.Price.Float64Value()
			itemTotalPrice, _ := item.TotalPrice.Float64Value()

			orderItems = append(orderItems, OrderItem{
				ProductID:  item.ProductID.String(),
				Price:      itemPrice.Float64,
				TotalPrice: itemTotalPrice.Float64,
				Quantity:   item.Quantity,
			})
		}

		id := o.ID.String()
		userId := o.UserID.String()
		tA, _ := o.TotalAmount.Float64Value()

		os = append(os, Order{
			OrderID:     &id,
			TotalAmount: &tA.Float64,
			Status:      (*string)(&o.Status),
			UserID:      &userId,
			ShippingAddress: Address{
				Line1:   shippingAddress.Line1,
				City:    shippingAddress.City,
				State:   shippingAddress.State,
				Country: shippingAddress.Country,
			},
			BillingAddress: Address{
				Line1:   billingAddress.Line1,
				City:    billingAddress.City,
				State:   billingAddress.State,
				Country: billingAddress.Country,
			},
			Items: orderItems,
		})
	}

	return os, nil
}
