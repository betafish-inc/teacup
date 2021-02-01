# teacup

Teacup provides tooling for quickly building microservices that operate in a consistent, managed environment.

## Workflow

To create a new Teacup microservice the normal work flow involves:

1. Create a new git repo.
2. Run `go mod init` in project root
3. Run `go get github.com/betafish-inc/teacup` to add the teacup dependency.
4. Create go launcher in `_apps/service/main.go` - using a standard naming allows tools to automatically build.
5. If using GitHub, add a GitHub Actions Workflow in `.github/workflows/teacup.yaml` using the template from `_apps/tools/teacup/teacup.yaml`.
6. Implement your service in service specific go packages in the project.

> TODO it would be nice to add a tool as part of the Teacup project to automate the lion's share of these tasks. e.g. `teacup init`.

## Building

Install [Go](https://golang.org) and in the project root run `go get -v -t -d ./...` to fetch all go dependencies.

Typical Go development process:

* Unit tests: `go test`
* Run: `go run _apps/subscriber/main.go`
* Build: `go build -o dist/subscriber _apps/subscriber/main.go`

## Runtime Environment

A Teacup based microservice operates in a standardized environment. Typically, this environment will run within a Docker Swarm or Kubernetes cluster. However, the same environment can also be run on bare OS's, Nomad clusters, etc.

In particular, Teacup assumes the following services are available:

* [NATS.io](https://nats.io/) for control plane and messaging
* [Redis](https://redis.io/) for data
* [PostgreSQL](https://postgresql.org) for SQL data

Configuration for accessing these services will be searched in the following order:

1. Environmental variables.
2. DNS SRV records.
4. Default ports on localhost.

> DNS SRV: This is most common in [Consul](https://consul.io/) managed environments. DNS SRV records must resolve using the default operating system DNS resolution system for Teacup microservices to lookup the information required.

For local development, run all the services your microservice relies on using default ports. The easiest is to install [Docker](https://docker.io) and create a file named `docker-compose.yml` containing the following:

```yaml
version: '3'
services:
  nats:
    image: nats:latest
    ports:
      - "6222:6222"
  redis:
    image: redislabs/redistimeseries
    ports:
      - "6379:6379"
```

Then run `docker-compose up` to start the services. Once the docker containers are running `go run _apps/subscriber/main.go` to run the subscriber microservice example. You'll need to send data to the same subject using `go run _apps/publisher/main.go`.

## NATS.io

Teacup supports NATS.io "core" servers running in an optional cluster and will search for the following configuration options: 

1. `NATS_ADDR` environmental variable. Should contain a comma separated list of host and port pairs e.g. `100.0.1.1:6222,100.0.1.2:6222`.
2. `nats.service.consul` & `nats.local ` DNS SRV entry.
4. `localhost:6222` fall back to the most likely default configuration.

## Redis

Teacup will attempt to connect to Redis using a connection URL:

* `REDIS_URL` environmental variable. Should be in the following format: `redis://[:password@]host:port/db`.

If the connection URL environmental variable doesn't exist, Teacup will search the following locations for Redis connection configuration information:

1. `REDIS_ADDR` environmental variable. Should contain the host and port like `localhost:6379`.
2. `redis.service.consul` & `redis.local` DNS SRV entry. 
4. `localhost:6379` fall back to the most likely default configuration.

Optionally, set a password and database using:

* `REDIS_PASSWORD` and `REDIS_DATABASE` environmental variables.

The default is no password, and database `0`.

## PostgreSQL

TBD

## Generating Mocks

Using Mockery:

Install mockery https://github.com/vektra/mockery
https://github.com/vektra/mockery#installation


From the root folder run:
```
mockery --name=ITeacup
```

Check the `mocks` folder output.