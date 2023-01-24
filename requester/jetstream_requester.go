package requester

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/ssd532/bench/v2"
)

// NATSRequesterFactory implements RequesterFactory by creating a Requester
// which publishes messages to NATS and waits to receive them.
type JetStreamRequesterFactory struct {
	URL                  string
	PayloadSize          int
	Stream               string
	AsyncPublish         bool
	MaxPublishAckPending int // wont' be used if async false
}

// GetRequester returns a new Requester, called for each Benchmark connection.
func (j *JetStreamRequesterFactory) GetRequester(num uint64) bench.Requester {
	return &jetstreamRequester{
		url:                  j.URL,
		payloadSize:          j.PayloadSize,
		stream:               strings.ToUpper(j.Stream + "-" + strconv.FormatUint(num, 10)),
		subject:              strings.ToUpper(j.Stream + "-" + strconv.FormatUint(num, 10) + ".subject"),
		asyncPublish:         j.AsyncPublish,
		maxPublishAckPending: j.MaxPublishAckPending,
	}
}

// natsRequester implements Requester by publishing a message to NATS and
// waiting to receive it.
type jetstreamRequester struct {
	url                  string
	payloadSize          int
	stream               string
	subject              string
	conn                 *nats.Conn
	js                   nats.JetStreamContext
	msg                  []byte
	sub                  *nats.Subscription
	inbound              chan nats.Msg
	asyncPublish         bool
	maxPublishAckPending int
}

// Setup prepares the Requester for benchmarking.
func (j *jetstreamRequester) Setup() error {
	conn, err := nats.Connect(j.url)
	if err != nil {
		return err
	}

	js, err := conn.JetStream(nats.PublishAsyncMaxPending(j.maxPublishAckPending))
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
		m.AckSync()
	}, nats.Durable("bench_consumer"))
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
	if j.asyncPublish {
		if _, err := j.js.PublishAsync(j.subject, j.msg); err != nil {
			return err
		}
	} else {
		if _, err := j.js.Publish(j.subject, j.msg); err != nil {
			return err
		}
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
	if j.asyncPublish {
		select {
		case <-j.js.PublishAsyncComplete():
		case <-time.After(5 * time.Second):
			return errors.New("timedout before recieving acks")
		}
	}
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
