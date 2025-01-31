package orders

// I need a db connection or something that implements it's interface
// I need queries
// I also need something that implements rabbit
// I think I want db access.

import (
	"context"
	"log"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/ponty96/my-proto-schemas/output/schemas"
	"google.golang.org/protobuf/proto"
)

func SetupTestDb(t *testing.T) *pgx.Conn {
	dbURL := "postgres://postgres:postgres@127.0.0.1:5432/simple-web-app-test?sslmode=disable"

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		t.Errorf("Unable to connect to database: %v\n", err)
	}
	return conn
}

// ---- rabbitmq.MQ Mock for Testing --- //
type MQMock struct {
	PublishedEvent []byte
	Closed         string
}

func (m *MQMock) Close() error {
	m.Closed = "closed"
	return nil
}

func (m *MQMock) Publish(ctx context.Context, o proto.Message) error {
	b, err := proto.Marshal(o)

	if err != nil {
		log.Panicf("failed to encode %s", err)
		return errors.Wrap(err, "failed to encode order proto")
	}
	m.PublishedEvent = b
	return nil
}

// --- End of rabbitmq.MQ Mock ---- //

func Test_NewOrderPublishSuccess(t *testing.T) {
	conn := SetupTestDb(t)
	ctx := context.Background()
	defer conn.Close(ctx)

	mockMQ := &MQMock{}
	defer mockMQ.Close()

	processer := NewProcessor(conn, mockMQ)

	sampleOrder := Order{
		OrderID:     stringPtr("ORD123456"),
		UserID:      stringPtr("USR789012"),
		TotalAmount: float64Ptr(159.97),
		Status:      stringPtr("PENDING"),
		Items: []OrderItem{
			{
				ProductID:  "PROD001",
				Quantity:   2,
				Price:      49.99,
				TotalPrice: 99.98,
			},
			{
				ProductID:  "PROD002",
				Quantity:   1,
				Price:      59.99,
				TotalPrice: 59.99,
			},
		},
		ShippingAddress: Address{
			Line1:      "123 Main Street",
			City:       "New York",
			State:      "NY",
			PostalCode: "10001",
			Country:    "USA",
		},
		BillingAddress: Address{
			Line1:      "123 Main Street",
			City:       "New York",
			State:      "NY",
			PostalCode: "10001",
			Country:    "USA",
		},
	}

	processer.NewOrder(ctx, &sampleOrder)

	orderEvent := &schemas.Order{}

	if mockMQ.PublishedEvent == nil {
		t.Error("Failed to Publish New Order")
	}

	if err := proto.Unmarshal(mockMQ.PublishedEvent, orderEvent); err != nil {
		t.Errorf("failed to decode %s", err)
	}

}

// Helper functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
