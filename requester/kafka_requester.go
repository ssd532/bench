package requester

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ssd532/bench/v2"
)

// KafkaRequesterFactory implements RequesterFactory by creating a Requester
// which publishes messages to Kafka and waits to consume them.
type KafkaRequesterFactory struct {
	URLs        []string
	PayloadSize int
	Topic       string
	DoConsume   bool
	IsAsync     bool
}

// GetRequester returns a new Requester, called for each Benchmark connection.
func (k *KafkaRequesterFactory) GetRequester(num uint64) bench.Requester {
	return &kafkaRequester{
		urls:        k.URLs,
		payloadSize: k.PayloadSize,
		topic:       k.Topic + "-" + strconv.FormatUint(num, 10),
		doConsume:   k.DoConsume,
		isAsync:     k.IsAsync,
	}
}

// kafkaRequester implements Requester by publishing a message to Kafka and
// waiting to consume it.
type kafkaRequester struct {
	urls              []string
	payloadSize       int
	topic             string
	asyncProducer     sarama.AsyncProducer
	syncProducer      sarama.SyncProducer
	consumer          sarama.Consumer
	partitionConsumer sarama.PartitionConsumer
	msg               *sarama.ProducerMessage
	doConsume         bool
	isAsync           bool
}

// Setup prepares the Requester for benchmarking.
func (k *kafkaRequester) Setup() error {
	config := sarama.NewConfig()
	var err error
	var asyncProducer sarama.AsyncProducer
	var syncProducer sarama.SyncProducer
	if k.isAsync {
		asyncProducer, err = sarama.NewAsyncProducer(k.urls, config)
	} else {
		config.Producer.Return.Successes = true
		syncProducer, err = sarama.NewSyncProducer(k.urls, config)
	}

	if err != nil {
		return err
	}

	var consumer sarama.Consumer
	var partitionConsumer sarama.PartitionConsumer
	if k.doConsume {
		consumer, err = sarama.NewConsumer(k.urls, nil)
		if err != nil {
			if k.isAsync {
				asyncProducer.Close()
			} else {
				syncProducer.Close()
			}
			return err
		}

		partitionConsumer, err = consumer.ConsumePartition(k.topic, 0, sarama.OffsetNewest)
		if err != nil {
			if k.isAsync {
				asyncProducer.Close()
			} else {
				syncProducer.Close()
			}
			consumer.Close()
			return err
		}
	}

	if k.isAsync {
		k.asyncProducer = asyncProducer
	} else {
		k.syncProducer = syncProducer
	}
	k.consumer = consumer
	k.partitionConsumer = partitionConsumer
	msg := make([]byte, k.payloadSize)
	for i := 0; i < k.payloadSize; i++ {
		msg[i] = 'A' + uint8(rand.Intn(26))
	}
	k.msg = &sarama.ProducerMessage{
		Topic: k.topic,
		Value: sarama.ByteEncoder(msg),
	}

	return nil
}

// Request performs a synchronous request to the system under test.
func (k *kafkaRequester) Request() error {
	if k.isAsync {
		k.asyncProducer.Input() <- k.msg
	} else {
		_, _, err := k.syncProducer.SendMessage(k.msg)
		if err != nil {
			panic("Error sending message: " + err.Error())
		}
	}

	if k.doConsume {
		select {
		case <-k.partitionConsumer.Messages():
			return nil
		case <-time.After(30 * time.Second):
			return errors.New("requester: Request timed out receiving")
		}
	}
	return nil
}

// Teardown is called upon benchmark completion.
func (k *kafkaRequester) Teardown() error {
	if k.doConsume {
		if err := k.partitionConsumer.Close(); err != nil {
			return err
		}
		if err := k.consumer.Close(); err != nil {
			return err
		}
	}
	if k.isAsync {
		if err := k.asyncProducer.Close(); err != nil {
			return err
		}
	} else {
		if err := k.syncProducer.Close(); err != nil {
			return err
		}
	}
	k.partitionConsumer = nil
	k.consumer = nil
	k.asyncProducer = nil
	k.syncProducer = nil
	return nil
}
