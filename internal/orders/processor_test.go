package orders

// I need a db connection or something that implements it's interface
// I need queries
// I also need something that implements rabbit
// I think I want db access.

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ponty96/my-proto-schemas/output/schemas"
	"github.com/ponty96/simple-web-app/internal/db"
)

func SetupTestDb(t *testing.T) *pgx.Conn {
	dbURL := "postgres://postgres:postgres@127.0.0.1:5432/simple-web-app-test?sslmode=disable"

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		t.Errorf("Unable to connect to database: %v\n", err)
	}
	return conn
}

func Test_NewOrder(t *testing.T) {
	conn := SetupTestDb(t)
	ctx := context.Background()
	defer conn.Close(ctx)

	p := NewProcessor(conn)

	// prepare test by deleting records
	p.queries.DeleteOrderItems(ctx)
	p.queries.DeleteOrders(ctx)

	// orderId := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	userId := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	productId := pgtype.UUID{Bytes: [16]byte{3}, Valid: true}

	items := []*schemas.OrderItem{
		{
			Price:      10.99,
			ProductId:  productId.String(),
			Quantity:   2,
			TotalPrice: 21.98,
		},
	}

	o := schemas.Order{
		UserId:      userId.String(),
		Items:       items,
		OrderStatus: "pending",
		TotalAmount: 21.98,
		ShippingAddress: &schemas.Address{
			City:   "New York",
			State:  "NY",
			Street: "123 Main St",
		},
		BillingAddress: &schemas.Address{
			City:   "New York",
			State:  "NY",
			Street: "123 Main St",
		},
	}

	if err := p.NewOrder(ctx, &o); err != nil {
		t.Errorf("Expected successfully created order %s", err)
	}

	orders, err := p.queries.ListOrders(ctx, userId)

	if err != nil {
		t.Errorf("Expected a list of orders %s", err)
	}

	mo := orders[0]

	if mo.UserID != userId {
		t.Error("Expected User's order")
	}

	if mo.Status != db.OrderStatus(o.OrderStatus) {
		t.Errorf("Expected status to be %s, got %v", o.OrderStatus, mo.Status)
	}

	floatVal, _ := mo.TotalAmount.Float64Value()

	if floatVal.Float64 != o.TotalAmount {
		t.Errorf("Expected total amount to be %f, got %f", o.TotalAmount, floatVal.Float64)
	}

	mItems, err := p.queries.ListOrderItems(ctx, mo.ID)

	if err != nil {
		t.Error("Expected to return Order Items")
	}

	if len(mItems) != 1 {
		t.Error("Expected the Order to have one Order Item")
	}

	it := mItems[0]

	if it.OrderID != mo.ID {
		t.Errorf("Expected ID to be %v, got %v", mo.ID, it.OrderID)
	}

}
