package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/zako/systemd-lsp/internal/lsp"
	"github.com/zako/systemd-lsp/internal/systemd"
)

func main() {
	logger := log.New(os.Stderr, "systemd-lsp: ", log.LstdFlags)
	server := lsp.NewServer(systemd.NewCatalog(), logger)
	if err := serve(os.Stdin, os.Stdout, server); err != nil && !errors.Is(err, io.EOF) {
		logger.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}

type handler interface {
	Handle(json.RawMessage) ([]byte, bool)
}

func serve(in io.Reader, out io.Writer, h handler) error {
	reader := bufio.NewReader(in)
	for {
		payload, err := readMessage(reader)
		if err != nil {
			return err
		}
		response, ok := h.Handle(payload)
		if !ok {
			continue
		}
		if err := writeMessage(out, response); err != nil {
			return err
		}
	}
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header line %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid Content-Length %q", value)
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, errors.New("missing Content-Length header")
	}
	payload := make([]byte, contentLength)
	_, err := io.ReadFull(r, payload)
	return payload, err
}

func writeMessage(w io.Writer, payload []byte) error {
	_, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(payload), payload)
	return err
}
