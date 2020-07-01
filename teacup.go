package teacup

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-redis/redis/v8"
	consul "github.com/hashicorp/consul/api"
	vault "github.com/hashicorp/vault/api"
) //https://redis.uptrace.dev

// Teacup provides essential tools for writing microservices with minimum boilerplate. Create an empty
// Teacup and then register at least one Worker or Sub then call Teacup.Start() to begin running.
type Teacup struct {
	Queue        *Queue
	workers      []Worker
	consulClient *consul.Client
	vaultClient  *vault.Client
	redisClient  *redis.Client
	context      context.Context
	cancel       context.CancelFunc
}

// NewTeacup creates a new Teacup instance ready for use.
func NewTeacup() *Teacup {
	return &Teacup{Queue: &Queue{}}
}

// Worker allows microservices to run "forever" without having to worry about initializing an environment
// or properly responding to system inputs. The worker should watch the context done channel to know when
// to terminate.
type Worker interface {
	// Start the Worker and run until the provided context's done channel is closed. The provided Teacup
	// reference can be used to access other Teacup managed services.
	Start(ctx context.Context, t *Teacup)
}

// The WorkerFunc type is an adapter to allow ordinary functions to act as Workers.
// If f is a function with the appropriate signature, WorkerFunc(f) is a Worker that calls f.
type WorkerFunc func(ctx context.Context, t *Teacup)

// Start calls f(ctx, t).
func (f WorkerFunc) Start(ctx context.Context, t *Teacup) {
	f(ctx, t)
}

// CancelFunc is called to unregister or unsubscribe a worker or subscriber from Teacup.
type CancelFunc func()

// Register a worker for running. THe Worker's Start() method will be called some time after the Teacup.Start() method
// is called and should not return unless the provided context is done or a fatal error occurs.
func (t *Teacup) Register(worker Worker) CancelFunc {
	t.workers = append(t.workers, worker)
	return func() {} // TODO support cancelling
}

// Context returns the context associated with the main Teacup "thread".
func (t *Teacup) Context() context.Context {
	if t.context == nil {
		t.context, t.cancel = context.WithCancel(context.Background())
	}
	return t.context
}

// Option returns the configuration setting associated with a key name. The function searches the
// environment first and if not found, tries to obtain the value from Consul.
func (t *Teacup) Option(_ context.Context, key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if ok {
		return val, nil
	}

	// TODO implement Consul fallback
	return "", nil
}

// Secret returns a secret associated with a key name. Secret searches the environment first and if not
// found, tries to obtain the value from Vault.
func (t *Teacup) Secret(_ context.Context, key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if ok {
		return val, nil
	}

	// TODO implement Vault fallback
	return "", nil
}

// ServiceAddr searches for a service address `name` by checking for:
//
// * `NAME_ADDR` - an environmental variable
// * `name.service.consul` - a SRV record
// * `name` - a consul service (TBD)
// * `localhost:<port>` - a fallback assuming the service is on the default port
func (t *Teacup) ServiceAddr(ctx context.Context, name string, port int) string {
	addr, ok := os.LookupEnv(fmt.Sprintf("%s_ADDR", strings.ToUpper(name)))
	if ok {
		return addr
	}
	// Try Consul using DNS
	resolver := net.Resolver{}
	_, addresses, err := resolver.LookupSRV(ctx, "", "", fmt.Sprintf("%s.service.consul", name))
	if err != nil || len(addresses) == 0 {
		// Try service on default port
		return fmt.Sprintf("localhost:%d", port)
	} else {
		return addresses[0].Target + ":" + strconv.Itoa(int(addresses[0].Port))
	}
}

// Start teacup running as long as one or more Worker or Sub are registered.
// Start does not return until the microservice is ready to terminate.
func (t *Teacup) Start() {
	ctx := t.Context()
	for _, worker := range t.workers {
		go worker.Start(ctx, t)
	}

	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM) // use SIGHUP for something?
	go func() {
		/*sig := */ <-signals
		t.cancel() // cancel the context
		// TODO watch/wait for all workers to terminate?
		done <- true // allow main to terminate
	}()

	<-done
}
