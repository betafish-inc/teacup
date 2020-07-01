package teacup

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/nsqio/go-nsq"
)

// Queue abstracts the underlying message queue from microservices. We will use nsq for "native" deployments
// but want to have the freedom of easily using a platform provided queue service if available.
type Queue struct {
	subs      []Subscriber
	nsqConfig *nsq.Config
	t         *Teacup
	producer  *Producer
}

// Sub to an event topic and channel. The returned CancelFunc can be used to cancel the subscription.
func (q *Queue) Sub(ctx context.Context, topic, channel string, sub Subscriber) (CancelFunc, error) {
	q.subs = append(q.subs, sub)
	consumer, err := nsq.NewConsumer(topic, channel, q.config())
	if err != nil {
		return nil, err
	}
	consumer.SetLoggerLevel(nsq.LogLevelError)
	consumer.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		// TODO might be useful to have a setting to automatically ignore empty messages?
		// TODO might be useful to support automatically JSON decoding messages for subscribers?

		// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
		ctx := context.Background()
		return sub.Message(ctx, q.t, topic, channel, msg.Body)
	}))
	// Find nsqlookupd location(s)
	var addresses []string
	addr, ok := os.LookupEnv("NSQLOOKUPD_ADDR")
	if ok {
		addresses = append(addresses, addr)
	} else {
		// Try Consul using DNS
		resolver := net.Resolver{}
		_, services, err := resolver.LookupSRV(ctx, "", "", "nsqlookupd.service.consul")
		if err != nil || len(services) == 0 {
			// Try Consul by accessing the API
			// TODO actually use Consul to lookup the service
			// return errors.New("nsqlookupd not configured")
			addresses = append(addresses, "localhost:4161")
		} else {
			for i := range services {
				addresses = append(addresses, services[i].Target+":"+strconv.Itoa(int(services[i].Port)))
			}
		}
	}
	err = consumer.ConnectToNSQLookupds(addresses)
	return func() { consumer.Stop() }, err
}

// Producer provides a Producer ready for sending messages to queues.
func (q *Queue) Producer(ctx context.Context) (*Producer, error) {
	if q.producer != nil {
		return q.producer, nil
	}
	producer, err := nsq.NewProducer(q.t.ServiceAddr(ctx, "nsqd", 4150), q.config())
	if err == nil {
		q.producer = &Producer{producer: producer}
	}
	return q.producer, err
}

// config returns the configuration to use for creating consumers or producers.
func (q *Queue) config() *nsq.Config {
	// TODO need to consider what settings we want to allow to be set. Settings should all automatically pull from Consul
	if q.nsqConfig == nil {
		q.nsqConfig = nsq.NewConfig()
	}
	return q.nsqConfig
}

// Producer provides an abstract way to publish messages to queues. It follows the nsq implementation closely
// and on other platforms, many operations may be NOOPs.
type Producer struct {
	producer *nsq.Producer
}

// Ping causes the Producer to connect to it's configured queue (if not already
// connected) and send a `noop` command, returning any error that might occur.
//
// This method can be used to verify that a newly-created Producer instance is
// configured correctly, rather than relying on the lazy "connect on Publish"
// behavior of a Producer.
func (p *Producer) Ping(_ context.Context) error {
	return p.producer.Ping()
}

// Stop initiates a graceful stop of the Producer (permanent)
//
// NOTE: this blocks until completion
func (p *Producer) Stop() {
	p.producer.Stop()
}

// PublishAsync publishes a message body to the specified topic
// but does not wait for the response from the queue.
func (p *Producer) PublishAsync(_ context.Context, topic string, body []byte) error {
	return p.producer.PublishAsync(topic, body, nil)
}

// MultiPublishAsync publishes a slice of message bodies to the specified topic
// but does not wait for the response from the queue.
func (p *Producer) MultiPublishAsync(_ context.Context, topic string, body [][]byte) error {
	return p.producer.MultiPublishAsync(topic, body, nil)
}

// Publish synchronously publishes a message body to the specified topic, returning
// an error if publish failed
func (p *Producer) Publish(_ context.Context, topic string, body []byte) error {
	return p.producer.Publish(topic, body)
}

// MultiPublish synchronously publishes a slice of message bodies to the specified topic, returning
// an error if publish failed
func (p *Producer) MultiPublish(_ context.Context, topic string, body [][]byte) error {
	return p.producer.MultiPublish(topic, body)
}

// DeferredPublish synchronously publishes a message body to the specified topic
// where the message will queue at the channel level until the timeout expires, returning
// an error if publish failed
func (p *Producer) DeferredPublish(_ context.Context, topic string, delay time.Duration, body []byte) error {
	return p.producer.DeferredPublish(topic, delay, body)
}

// DeferredPublishAsync publishes a message body to the specified topic
// where the message will queue at the channel level until the timeout expires
// but does not wait for the response from the queue.
func (p *Producer) DeferredPublishAsync(topic string, delay time.Duration, body []byte) error {
	return p.producer.DeferredPublishAsync(topic, delay, body, nil)
}

// Subscriber allows microservices to process queue messages on a topic/channel without dealing directly
// with the underlying message queue (including all it's various settings and configuration options). Instead,
// microservices implement Subscriber and Sub to a topic/channel and receive messages as they are ready for processing.
type Subscriber interface {
	// Message is called for each received message from the given topic/channel combination.
	// Errors can be used to force a requeue of the message.
	// TODO we need better documentation on exactly how errors are used to force requeues (and any other actions)
	Message(ctx context.Context, t *Teacup, topic, channel string, msg []byte) error
}

// The SubscriberFunc type is an adapter to allow ordinary functions to act as Subscribers.
// If f is a function with the appropriate signature, SubscriberFunc(f) is a Subscriber that calls f.
type SubscriberFunc func(ctx context.Context, t *Teacup, topic, channel string, msg []byte) error

// Message calls f(cxt, topic, channel, msg).
func (f SubscriberFunc) Message(ctx context.Context, t *Teacup, topic, channel string, msg []byte) error {
	return f(ctx, t, topic, channel, msg)
}
