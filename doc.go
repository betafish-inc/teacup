// Package teacup provides simple tooling for rapidly building microservices that will operate in a standardized
// environment. That environment currently assumes:
//
//  * service discovery using Hashicorp Consul https://hashicorp.com/consul
//  * secrets using Hashicorp Vault https://hashicorp.com/vault
//  * message queues using nsq.io https://nsq.io
//  * data storage using Redis https://redis.io
package teacup
