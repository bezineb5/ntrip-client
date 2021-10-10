package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	ntrip_client "github.com/bezineb5/ntrip-client"
	"github.com/bezineb5/ntrip-client/input"
	"github.com/bezineb5/ntrip-client/output"
)

func main() {
	fmt.Println("Starting NTRIP client")

	f := os.Stdout
	writer := bufio.NewWriter(f)
	output := output.NewWriterOutput(writer)

	registry := ntrip_client.NewInMemoryRegistry()
	client := input.NewNtripV2Client("http://caster.centipede.fr:2101/")
	source := ntrip_client.NewSource(client)
	if err := source.RegisterMountPoints(registry); err != nil {
		fmt.Println(err)
		return
	}

	selector := ntrip_client.NewRegistrySelector(registry, 0.01)

	if err := selector.SetLocation(46.3531178, 0.54); err != nil {
		fmt.Println(err)
		return
	}

	ch, err := selector.Stream()
	if err != nil {
		fmt.Println(err)
		return
	}

	output.Stream(ch)

	waitForSignal()
}

func waitForSignal() {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received.
	s := <-c
	log.Println("Got signal:", s)
}
