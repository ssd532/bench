package requester

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/kishansairam9/bench/v2"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/message"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

// AMQPRequesterFactory implements RequesterFactory by creating a Requester
// which publishes messages to an AMQP exchange and waits to consume them.
type RMQStreamRequesterFactory struct {
	URLs        []string
	PayloadSize int
	Stream      string
}

// GetRequester returns a new Requester, called for each Benchmark connection.
func (r *RMQStreamRequesterFactory) GetRequester(num uint64) bench.Requester {
	return &rmqstreamRequester{
		urls:        r.URLs,
		payloadSize: r.PayloadSize,
		stream:      r.Stream + "-" + strconv.FormatUint(num, 10),
	}
}

// amqpRequester implements Requester by publishing a message to an AMQP
// exhcnage and waiting to consume it.
type rmqstreamRequester struct {
	urls        []string
	payloadSize int
	stream      string
	producer    *stream.Producer
	consumer    *stream.Consumer
	msg         *amqp.AMQP10
	inbound     chan amqp.Message
	env         *stream.Environment
}

// Setup prepares the Requester for benchmarking.
func (r *rmqstreamRequester) Setup() error {
	env, err := stream.NewEnvironment(
		stream.NewEnvironmentOptions().
			SetHost("localhost").
			SetPort(5552).
			SetUser("guest").
			SetPassword("guest"))
	r.env = env
	if err != nil {
		return err
	}
	err = env.DeclareStream(r.stream,
		&stream.StreamOptions{
			MaxLengthBytes: stream.ByteCapacity{}.GB(2),
		},
	)
	if err != nil {
		return err
	}
	producer, err := env.NewProducer(r.stream, stream.NewProducerOptions().SetBatchSize(1))
	if err != nil {
		return err
	}

	r.inbound = make(chan amqp.Message)
	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		// fmt.Printf("Sent 1 msg\n")
		r.inbound <- *message
	}

	consumer, err := env.NewConsumer(
		r.stream,
		handleMessages,
		stream.NewConsumerOptions().
			SetConsumerName("my_consumer"). // set a consumer name
			SetOffset(stream.OffsetSpecification{}.First()))
	if err != nil {
		return err
	}
	r.producer = producer
	r.consumer = consumer
	msg := make([]byte, r.payloadSize)
	for i := 0; i < r.payloadSize; i++ {
		msg[i] = 'A' + uint8(rand.Intn(26))
	}
	r.msg = amqp.NewMessage(msg)
	return nil
}

// Request performs a synchronous request to the system under test.
func (r *rmqstreamRequester) Request() error {
	if err := r.producer.BatchSend([]message.StreamMessage{r.msg}); err != nil {
		return err
	}
	select {
	case <-r.inbound:
		// fmt.Printf("Recieved 1\n")
	case <-time.After(30 * time.Second):
		return errors.New("requester: Request timed out receiving")
	}
	return nil
}

// Teardown is called upon benchmark completion.
func (r *rmqstreamRequester) Teardown() error {
	if err := r.producer.Close(); err != nil {
		return err
	}
	if err := r.consumer.Close(); err != nil {
		return err
	}
	if err := r.env.DeleteStream(r.stream); err != nil {
		return err
	}
	if err := r.env.Close(); err != nil {
		return err
	}
	r.consumer = nil
	r.producer = nil
	r.env = nil
	return nil
}
