package example

import (
	"context"
	"encoding/hex"
	"log"

	"github.com/betafish-inc/teacup"
)

// Echo pulls messages from nsq and prints the results to stdout.
// Echo demonstrates a normal, teacup message driven microservice.
type Echo struct {
	MessagesAreBinary bool // set to true if messages are binary and should only be hex dumped
}

// Message handles incoming messages by printing them to stdout.
func (e *Echo) Message(_ context.Context, _ *teacup.Teacup, topic, channel string, msg []byte) error {

	if e.MessagesAreBinary {
		log.Println("topic:", topic, "channel:", channel, "data:", hex.Dump(msg))
	} else {
		// We assume messages JSON strings because we tend to pass JSON encoded strings.
		// We could add the ability to pretty print the JSON...
		log.Println("topic:", topic, "channel:", channel, "data:", string(msg))
	}

	// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
	return nil
}
