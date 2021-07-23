package requester

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/kishansairam9/bench/v2"
	"github.com/nats-io/nats.go"
)

// NATSRequesterFactory implements RequesterFactory by creating a Requester
// which publishes messages to NATS and waits to receive them.
type JetStreamRequesterFactory struct {
	URL         string
	PayloadSize int
	Stream      string
}

// GetRequester returns a new Requester, called for each Benchmark connection.
func (j *JetStreamRequesterFactory) GetRequester(num uint64) bench.Requester {
	return &jetstreamRequester{
		url:         j.URL,
		payloadSize: j.PayloadSize,
		stream:      strings.ToUpper(j.Stream + "-" + strconv.FormatUint(num, 10)),
		subject:     strings.ToUpper(j.Stream + "-" + strconv.FormatUint(num, 10) + ".subject"),
	}
}

// natsRequester implements Requester by publishing a message to NATS and
// waiting to receive it.
type jetstreamRequester struct {
	url         string
	payloadSize int
	stream      string
	subject     string
	conn        *nats.Conn
	js          nats.JetStreamContext
	msg         []byte
	sub         *nats.Subscription
	inbound     chan nats.Msg
}

// Setup prepares the Requester for benchmarking.
func (j *jetstreamRequester) Setup() error {
	conn, err := nats.Connect(j.url)
	if err != nil {
		return err
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return err
	}

	_, err = js.AddStream(&nats.StreamConfig{Name: j.stream, Subjects: []string{j.subject}})
	if err != nil {
		conn.Close()
		return err
	}

	j.inbound = make(chan nats.Msg)
	sub, err := js.Subscribe(j.subject, func(m *nats.Msg) {
		j.inbound <- *m
	})
	if err != nil {
		j.inbound = nil
		conn.Close()
		return err
	}

	j.conn = conn
	j.js = js
	j.sub = sub
	msg := make([]byte, j.payloadSize)
	for i := 0; i < j.payloadSize; i++ {
		msg[i] = 'A' + uint8(rand.Intn(26))
	}
	j.msg = msg
	return nil
}

// Request performs a synchronous request to the system under test.
func (j *jetstreamRequester) Request() error {
	if _, err := j.js.PublishAsync(j.subject, j.msg); err != nil {
		return err
	}
	select {
	case <-j.inbound:
	case <-time.After(30 * time.Second):
		return errors.New("timeout")
	}
	return nil
}

// Teardown is called upon benchmark completion.
func (j *jetstreamRequester) Teardown() error {
	if err := j.sub.Unsubscribe(); err != nil {
		return err
	}
	if err := j.js.DeleteStream(j.stream); err != nil {
		return err
	}
	j.sub = nil
	j.conn.Close()
	j.conn = nil
	return nil
}
