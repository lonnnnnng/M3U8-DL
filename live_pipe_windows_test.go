//go:build windows

package main

import (
	"io"
	"os"
	"testing"
	"time"
)

func TestCreateWindowsLivePipeConnectsClient(t *testing.T) {
	name, err := randomLivePipeName()
	if err != nil {
		t.Fatal(err)
	}
	server, path, err := createLivePipe(name, livePipeEnv{})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	connected := make(chan error, 1)
	go func() {
		connected <- connectLivePipe(server)
	}()

	client, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	select {
	case err := <-connected:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("named pipe client did not connect")
	}

	readDone := make(chan []byte, 1)
	go func() {
		got, _ := io.ReadAll(client)
		readDone <- got
	}()
	if _, err := server.Write([]byte("pipe-ok")); err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if got := string(<-readDone); got != "pipe-ok" {
		t.Fatalf("windows named pipe payload mismatch: %q", got)
	}
}
