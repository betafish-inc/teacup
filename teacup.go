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
) //https://redis.uptrace.dev

// Teacup provides essential tools for writing microservices with minimum boilerplate. Create an empty
// Teacup and then register at least one Worker or Sub then call Teacup.Start() to begin running.
type Teacup struct {
	workers      []*WorkerReg
	redisClient  *redis.Client
	natsDone 	 chan bool
	done chan bool
	context      context.Context
	cancel       context.CancelFunc
	queue *Queue
}

// NewTeacup creates a new Teacup instance ready for use.
func NewTeacup() *Teacup {
	return &Teacup{}
}

// Queue returns a queue helper for interacting with the message bus.
func (t *Teacup) Queue() *Queue {
	if t.queue == nil {
		t.queue = &Queue{t:t}
	}
	return t.queue
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

// WorkerReg contains information about a Worker registration with Teacup.
type WorkerReg struct {
	Worker Worker
	cancel context.CancelFunc
}

// Register a worker for running. THe Worker's Start() method will be called some time after the Teacup.Start() method
// is called and should not return unless the provided context is done or a fatal error occurs.
func (t *Teacup) Register(worker Worker) {
	t.workers = append(t.workers, &WorkerReg{Worker:worker})
}

// Context returns the context associated with the main Teacup goroutine.
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
// * `NAME_ADDR` - an environmental variable containing a host:port pair
// * `name.service.consul` & `name.local` - a SRV record
// * `localhost:<port>` - a fallback assuming the service is on the default port
func (t *Teacup) ServiceAddr(ctx context.Context, name string, port int) string {
	addr, ok := os.LookupEnv(fmt.Sprintf("%s_ADDR", strings.ToUpper(name)))
	if ok {
		return addr
	}
	// Check for DNS SRV records
	resolver := net.Resolver{}
	_, addresses, err := resolver.LookupSRV(ctx, "", "", fmt.Sprintf("%s.service.consul", name))
	if err == nil && len(addresses) > 0 {
		return addresses[0].Target + ":" + strconv.Itoa(int(addresses[0].Port))
	}
	_, addresses, err = resolver.LookupSRV(ctx, "", "", fmt.Sprintf("%s.local", name))
	if err == nil && len(addresses) > 0 {
		return addresses[0].Target + ":" + strconv.Itoa(int(addresses[0].Port))
	}
	// Try service on default port
	return fmt.Sprintf("localhost:%d", port)
}


// ServiceAddr searches for a service address `name` by checking for:
//
// * `NAME_ADDR` - an environmental variable containing comma separated host:port pairs.
// * `name.service.consul` & `name.local` - one or more SRV records
// * `localhost:<port>` - a fallback assuming the service is on the default port
func (t *Teacup) ServiceAddrs(ctx context.Context, name string, port int) []string {
	addr, ok := os.LookupEnv(fmt.Sprintf("%s_ADDR", strings.ToUpper(name)))
	if ok {
		return strings.Split(addr, ",")
	}
	// Check for DNS SRV records
	resolver := net.Resolver{}
	_, addresses, err := resolver.LookupSRV(ctx, "", "", fmt.Sprintf("%s.service.consul", name))
	if err == nil && len(addresses) > 0 {
		found:=make([]string, len(addresses))
		for i, a := range addresses {
			found[i] = a.Target + ":" + strconv.Itoa(int(a.Port))
		}
		return found
	}
	_, addresses, err = resolver.LookupSRV(ctx, "", "", fmt.Sprintf("%s.local", name))
	if err == nil && len(addresses) > 0 {
		found:=make([]string, len(addresses))
		for i, a := range addresses {
			found[i] = a.Target + ":" + strconv.Itoa(int(a.Port))
		}
		return found
	}
	// Try service on default port
	return []string{fmt.Sprintf("localhost:%d", port)}
}

// Start teacup running as long as one or more Worker or Sub are registered.
// Start does not return until the microservice is ready to terminate.
func (t *Teacup) Start() {
	ctx := t.Context()
	for _, reg := range t.workers {
		sub, cancelFunc := context.WithCancel(ctx)
		reg.cancel = cancelFunc
		go reg.Worker.Start(sub, t)
	}

	signals := make(chan os.Signal, 1)
	t.done = make(chan bool, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM) // use SIGHUP for something?
	go func() {
		/*sig := */ <-signals
		t.Stop()
	}()

	<-t.done
}

// Stop teacup from running.
func (t *Teacup) Stop() {
	t.cancel() // cancel the context and all child context which should stop all the workers
	// TODO how can we wait for all the workers to finish shutting down?
	if t.queue != nil {
		_ = t.queue.Client.Drain()
		<-t.natsDone // Wait for client to drain (should probably put a limit on this so we don't wait forever)
	}
	t.done <- true // allow main to terminate
}