package requester

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/kishansairam9/bench/v2"
	lift "github.com/liftbridge-io/go-liftbridge/v2"
)

// AMQPRequesterFactory implements RequesterFactory by creating a Requester
// which publishes messages to an AMQP exchange and waits to consume them.
type LiftbridgeRequesterFactory struct {
	URLs        []string
	PayloadSize int
	Stream      string
}

// GetRequester returns a new Requester, called for each Benchmark connection.
func (r *LiftbridgeRequesterFactory) GetRequester(num uint64) bench.Requester {
	return &liftbridgeRequester{
		urls:        r.URLs,
		payloadSize: r.PayloadSize,
		subject:     r.Stream + "-" + strconv.FormatUint(num, 10),
		stream:      r.Stream + "-" + strconv.FormatUint(num, 10) + "-stream",
	}
}

// amqpRequester implements Requester by publishing a message to an AMQP
// exhcnage and waiting to consume it.
type liftbridgeRequester struct {
	urls        []string
	payloadSize int
	stream      string
	subject     string
	client      lift.Client
	inbound     chan lift.Message
	errch       chan error
	msg         []byte
}

// Setup prepares the Requester for benchmarking.
func (l *liftbridgeRequester) Setup() error {
	client, err := lift.Connect(l.urls)
	if err != nil {
		return err
	}
	if err := client.CreateStream(context.Background(), l.subject, l.stream); err != nil {
		if err != lift.ErrStreamExists {
			return err
		}
	}

	l.inbound = make(chan lift.Message)
	l.errch = make(chan error)
	handleMessages := func(msg *lift.Message, err error) {
		if err != nil {
			l.errch <- err
		}
		l.inbound <- *msg
	}

	if err := client.Subscribe(context.Background(), l.stream, handleMessages, lift.StartAtEarliestReceived()); err != nil {
		l.inbound = nil
		l.errch = nil
		return err
	}

	l.client = client
	msg := make([]byte, l.payloadSize)
	for i := 0; i < l.payloadSize; i++ {
		msg[i] = 'A' + uint8(rand.Intn(26))
	}
	l.msg = msg
	return err
}

// Request performs a synchronous request to the system under test.
func (l *liftbridgeRequester) Request() error {
	if err := l.client.PublishAsync(context.Background(), l.stream, l.msg,
		func(ack *lift.Ack, err error) {
			if err != nil {
				l.errch <- err
			}
		}); err != nil {
		return err
	}
	select {
	case <-l.inbound:
	case err := <-l.errch:
		return err
	case <-time.After(30 * time.Second):
		return errors.New("requester: Request timed out receiving")
	}
	return nil
}

// Teardown is called upon benchmark completion.
func (l *liftbridgeRequester) Teardown() error {
	err := l.client.Close()
	if err != nil {
		return err
	}
	l.client = nil
	return nil
}
