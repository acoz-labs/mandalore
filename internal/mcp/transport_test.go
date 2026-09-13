package memorymcp

import (
	"bufio"
	"context"
	"github.com/acoz-labs/mandalore/internal/api"
	"io"
	"strings"
	"testing"
	"time"
)

func TestFrameSizeBound(t *testing.T) {
	input := io.NopCloser(strings.NewReader(strings.Repeat("x", MaxFrameBytes) + "\n"))
	r := &framedReader{source: input, reader: bufio.NewReaderSize(input, MaxFrameBytes+1)}
	if _, err := r.Read(make([]byte, 128)); err == nil {
		t.Fatal("unbounded frame accepted")
	}
}

func TestCancellationClosesBlockedTransport(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, api.New(nil, true), reader, io.Discard) }()
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled server retained a blocked reader")
	}
}

func TestTransportEOFClean(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := Run(ctx, api.New(nil, true), io.NopCloser(strings.NewReader("")), io.Discard); err != nil {
		t.Fatal(err)
	}
}
