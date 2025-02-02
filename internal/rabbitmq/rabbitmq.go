package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/ponty96/my-proto-schemas/output/schemas"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrRetryable = errors.New("retryable error")

type MQ interface {
	Close() error
	Publish(context.Context, proto.Message) error
}

type Config struct {
	URL             string
	ConnectionCount int
}

// TODO: Consider reconnection logic and multiple connections.
func NewRabbitMQ(cfg Config) *RabbitMQ {
	conn, err := amqp.Dial(cfg.URL)
	failOnError(err, "Failed to connect to RabbitMQ")

	return &RabbitMQ{
		conn: conn,
	}
}

type RabbitMQ struct {
	// consider a different connect for publisher and consumer
	conn *amqp.Connection
	done chan struct{}
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
		return errors.Wrap(err, "publish: failed to open a channel")
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
		return errors.Wrap(err, "publish: failed to declare an exchange")
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

func (r *RabbitMQ) Consume(ctx context.Context, in proto.Message, f func(ctx context.Context, o proto.Message) error) error {
	m := r.GetMessageMeta(in)

	ch, err := r.conn.Channel()
	if err != nil {
		return errors.Wrap(err, "consume: failed to open a channel")
	}

	q, err := ch.QueueDeclare(
		m.msgType, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		return errors.Wrap(err, "consume: failed to declare a queue")
	}

	err = ch.QueueBind(
		q.Name,          // queue name
		m.msgRoutingKey, // routing key
		m.msgExchange,   // exchange
		false,
		nil,
	)

	if err != nil {
		return errors.Wrap(err, "consume: failed to bind queue")
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	if err != nil {
		return errors.Wrap(err, "consume: failed to register a consumer")
	}

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case <-r.done:
				return
			case d, ok := <-msgs:
				if !ok {
					log.Print("I got NO message from the consumer")
					return
				}
				log.Print("I got a message from the consumer")
				event := proto.Clone(in)

				if err := proto.Unmarshal(d.Body, event); err != nil {
					log.Errorf("failed to decode %+v with err %s", d.Body, err)
				}
				ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				if err := f(ctx, event); err != nil {
					log.Errorf("EventConsumer: %s", err)
					// Decide whether to requeue based on error type
					if errors.Is(err, ErrRetryable) {
						d.Nack(false, true) // requeue
					} else {
						d.Nack(false, false) // don't requeue
					}
				}

				d.Ack(false)
			}
		}
	}()

	return nil
}
