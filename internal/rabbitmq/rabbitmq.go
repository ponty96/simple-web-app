package rabbitmq

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/ponty96/my-proto-schemas/output/schemas"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MQ interface {
	Close() error
	Publish(context.Context, proto.Message) error
}

type Config struct {
	URL             string
	ConnectionCount int
}

// TODO: Consider reconnection logic and multiple connections.
func NewRabbitMQ(cfg Config) MQ {
	conn, err := amqp.Dial(cfg.URL)
	failOnError(err, "Failed to connect to RabbitMQ")

	return &RabbitMQ{
		conn: conn,
	}
}

type RabbitMQ struct {
	// consider a different connect for publisher and consumer
	conn *amqp.Connection
}

func (e *RabbitMQ) Close() error {
	return e.conn.Close()
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

type Meta struct {
	msgType       string
	msgRoutingKey string
	msgExchange   string
}

func (e *RabbitMQ) GetMessageMeta(msg proto.Message) Meta {
	descriptor := msg.ProtoReflect().Descriptor()
	options := descriptor.Options()

	// Get the extension values
	msgType := proto.GetExtension(options, schemas.E_MsgType).(string)
	msgRoutingKey := proto.GetExtension(options, schemas.E_MsgRoutingKey).(string)
	msgExchange := proto.GetExtension(options, schemas.E_MsgExchange).(string)

	// Now you can use these values
	fmt.Printf("Message Type: %s\n", msgType)
	fmt.Printf("Routing Key: %s\n", msgRoutingKey)
	fmt.Printf("Exchange: %s\n", msgExchange)
	return Meta{msgType, msgRoutingKey, msgExchange}
}

func (r *RabbitMQ) Publish(ctx context.Context, o proto.Message) error {
	m := r.GetMessageMeta(o)

	ch, err := r.conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open a channel")
	}
	defer ch.Close()

	if err = ch.ExchangeDeclare(
		m.msgExchange, // name
		"fanout",      // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	); err != nil {
		return errors.Wrap(err, "failed to declare an exchange")
	}

	b, err := proto.Marshal(o)

	if err != nil {
		log.Panicf("failed to encode %s", err)
		return errors.Wrap(err, "failed to encode order proto")
	}
	if err = ch.PublishWithContext(ctx,
		m.msgExchange,   // exchange
		m.msgRoutingKey, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(b),
		}); err != nil {
		return errors.Wrap(err, "failed to publish order")
	}
	return nil
}
