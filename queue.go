package teacup

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
)

// Queue abstracts the underlying message queue from microservices. We will use nsq for "native" deployments
// but want to have the freedom of easily using a platform provided queue service if available.
type Queue struct {
	subs      []Subscriber
	Client   *nats.Conn
	t         *Teacup
	producer  *Producer
}

// Sub to an event topic and channel. The returned CancelFunc can be used to cancel the subscription.
func (q *Queue) Sub(ctx context.Context, topic, channel string, sub Subscriber) {
	q.subs = append(q.subs, sub)
	// TODO is there anything we can do with the error
	_, _ = q.client(ctx).QueueSubscribe(topic, channel, func(msg *nats.Msg) {
		// TODO is there anything we can do with this error
		thisCtx := context.WithValue(q.t.Context(), "reply", msg.Reply)
		thisCtx = context.WithValue(thisCtx, "subject", msg.Subject)
		_ = sub.Message(thisCtx, q.t, topic, channel, msg.Data)
	})
}

// Producer provides a Producer ready for sending messages to queues.
func (q *Queue) Producer(ctx context.Context) *Producer {
	if q.producer != nil {
		return q.producer
	}
	q.producer = &Producer{client: q.client(ctx)}
	return q.producer
}

// client returns a valid NATS client connection.
func (q *Queue) client(ctx context.Context) *nats.Conn {
	if q.Client == nil {
		q.t.natsDone = make(chan bool, 1)
		servers := q.t.ServiceAddrs(ctx, "nats", 4222)
		addrs := make([]string, len(servers))
		for i, s := range servers {
			addrs[i] = "nats://"+s
		}
		opts := nats.Options{
			Servers: addrs,
			ClosedCB: func(_ *nats.Conn) {
				q.t.natsDone<-true
			},
		}
		conn, err := opts.Connect()
		// TODO do something smarter with the error
		if err != nil {
			log.Fatal("Could not connect", err)
		}
		q.Client = conn
	}
	return q.Client
}

// Producer provides an abstract way to publish messages to queues. It follows the nsq implementation closely
// and on other platforms, many operations may be NOOPs.
type Producer struct {
	client *nats.Conn
}

// Publish synchronously publishes a message body to the specified topic, returning
// an error if publish failed
func (p *Producer) Publish(_ context.Context, topic string, body []byte) error {
	return p.client.Publish(topic, body)
}

// Request synchronously publishes a message request to the specified topic and waits for a response.
func (p *Producer) Request(ctx context.Context, topic string, body []byte) ([]byte, error) {
	response, err := p.client.RequestWithContext(ctx, topic, body)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

// Subscriber allows microservices to process queue messages on a topic/channel without dealing directly
// with the underlying message queue (including all it's various settings and configuration options). Instead,
// microservices implement Subscriber and Sub to a topic/channel and receive messages as they are ready for processing.
type Subscriber interface {
	// Message is called for each received message from the given topic/channel combination.
	// The "reply" channel subject is added to the context for message handlers that need to respond to
	// a message queue request.
	//
	// TODO make the following an actual go doc example
	//
	// Example:
	//
	// Message(ctx context.Context, t *Teacup, topic, channel string, msg []byte) error {
	//   t.Publisher(ctx).Publish(ctx, ctx.Value("reply"), []byte("{\"foo\":\"bar\"}"))
	// }
	//
	//
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
