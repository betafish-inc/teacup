package main

import (
	"github.com/betafish-inc/teacup"
	"github.com/betafish-inc/teacup/example"
)

func main() {
	// the job of main for most Teacup microservices should be to parse command line, perform minimal
	// configuration and then launch Teacup. Most microservices don't need command line
	t := teacup.NewTeacup()
	_, _ = t.Queue.Sub(t.Context(), "example", "echo", &example.Echo{})
	t.Start()
}
