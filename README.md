# teacup

Teacup provides tooling for quickly building microservices that operate in a consistent, managed environment.

## Building

Install [Go](https://golang.org) and in the project root run `go get -v -t -d ./...` to fetch all go dependencies.

Typical Go development process:

* Unit tests: `go test`
* Run: `go run _apps/subscriber/main.go`
* Build: `go build -o dist/subscriber _apps/subscriber/main.go`

TODO unpin vault go.mod (and use normal versioning) once Vault API has updated its module support. Pinned using https://github.com/hashicorp/vault/issues/9072 to commit.

## Runtime Environment

A Teacup based microservice will operate in a common standardized environment. Typically, this environment will run within a Kubernetes cluster. However, the same environment can also be run on bare OS's, Docker clusters, etc.

In particular, Teacup assumes the following service are available:

* Redis for data
* A message queue (currently `nsq`) messaging
* Hashicorp Consul for configuration
* Hashicorp Vault for secrets
* PostgreSQL for SQL data

Configuration for accessing these services will be search for in the following order:

1. Environmental variables
2. Consul DNS. Registering each infrastructure service as a Consul service will automatically create the DNS entries required. Consul DNS must resolve using the default operating system DNS resolution system for Teacup microservices to lookup the information required.
3. Consul service via Consul API (not implemented yet)
4. Default, localhost

For local development, simply run all the services your microservice relies on using default ports. For example, the subscriber example can be run locally by installing redis and nsq:

Mac (using [Homebrew](https://brew.sh)): `brew install redis nsq`

Run the following on separate terminal sessions:

* `redis-server` (optional) to run Redis on its default port 6379.
* `nsqlookupd` to run the nsq lookup service on its default port 4160.
* `nsqd --nsqlookupd-tcp-addr localhost:4160` to run the nsq service on its default port of 4150.
* `nsqadmin --nsqlookupd-tcp-addr localhost:4160` (optional) to run the nsq admin web UI for easier troubleshooting of queue issues.
* `cat ./data.json | to_nsq -topic example --nsqd-tcp-address localhost:4150` (optional) to send a json message from a file to the `example` topic (see note below).
* `go run _apps/subscriber/main.go` to run the subscriber microservice example.
* `cat ./data.json | to_nsq -topic example --nsqd-tcp-address localhost:4150` to send json from a file to the `example` topic.

> Note: nsq creates topics on demand and nsqlookupd will propagate the queue information to subscribers. If the queue doesn't exist, there may be harmless error messages from subscribers (indicating the topic can't be found). It is normal for a short delay (typically 1-2 seconds) between the initial message posted to a topic and for subscribers to receive the first message. Please read the [nsq docs](https://nsq.io) for more details.

## NSQ

The [nsq](https://nsq.io) message queue requires two services to be running: nsqlookupd and nsqd. 

### nsqlookupd

Teacup will search the following locations for nsqlookupd configuration information.

1. `NSQLOOKUPD_ADDR` environmental variable. Should contain the host and port like `localhost:4161`.
2. `nsqlookupd.service.consul` DNS SRV entry.
3. `nsqlookupd` service using the Consul API.
4. `localhost:4161` fall back to the most likely default configuration.

### nsqd

Teacup will search the following locations for nsqd configuration information.

1. `NSQD_ADDR` environmental variable. Should contain the host and port like `localhost:4150`.
2. `nsqd.service.consul` DNS SRV entry. 
3. `nsqd` service using the Consul API.
4. `localhost:4150` fall back to the most likely default configuration.

## Consul

Teacup will search the following locations for Consul configuration information:

1. `CONSUL_ADDR` environmental variable. Should contain the host and port like `localhost:8500`.
2. `consul.service.consul` DNS SRV entry. 
3. `localhost:8500` fall back to the most likely default configuration.

## Vault

Teacup will search the following locations for Vault address configuration information:

1. `CONSUL_ADDR` environmental variable. Should contain the host and port like `localhost:8200`.
2. `vault.service.consul` DNS SRV entry. 
3. `vault` service using the Consul API.
4. `localhost:8200` fall back to the most likely default configuration.

Vault authentication uses AppRole and requires the following environmental variables:

* "VAULT_ROLE_ID"
* "VAULT_SECRET_ID" 

## Redis

Teacup will attempt to connect to Redis using a connection URL:

* `REDIS_URL` environmental variable. Should be in the following format: `redis://[:password@]host:port/db`.

If the connection URL doesn't exist, Teacup will search the following locations for Redis connection configuration information:

1. `REDIS_ADDR` environmental variable. Should contain the host and port like `localhost:6379`.
2. `redis.service.consul` DNS SRV entry. 
3. `redis` service using the Consul API.
4. `localhost:6379` fall back to the most likely default configuration.

Optionally, set a password and database using:

* `REDIS_PASSWORD` and `REDIS_DATABASE` environmental variables.

> TODO we should probably grab these secrets from Vault as a fallback

## PostgreSQL

TBD