// Package main provides the transceiverctl binary.
package main

import (
	"os"

	"github.com/vexxhost/transceiver_exporter/pkg/transceiverctl"
)

func main() {
	os.Exit(transceiverctl.Run(os.Args[1:], os.Stdout, os.Stderr))
}
