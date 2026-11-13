package main

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	"strconv"
	"testing"
)

func TestReadAndWriteMessage(t *testing.T) {
	input := bytes.NewBufferString("Content-Length: 15\r\n\r\n{\"jsonrpc\":\"2\"}")
	payload, err := readMessage(bufio.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"jsonrpc":"2"}` {
		t.Fatalf("payload = %q", payload)
	}

	var output bytes.Buffer
	if err := writeMessage(&output, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if output.String() != "Content-Length: 11\r\n\r\n{\"ok\":true}" {
		t.Fatalf("framed message = %q", output.String())
	}
}

func TestServeWritesHandlerResponse(t *testing.T) {
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	input := bytes.NewBufferString("Content-Length: " + strconv.Itoa(len(request)) + "\r\n\r\n" + string(request))
	var output bytes.Buffer
	err := serve(input, &output, staticHandler{responses: [][]byte{[]byte(`{"jsonrpc":"2.0","id":1,"result":null}`)}})
	if err == nil {
		t.Fatal("serve returned nil, want EOF after one message")
	}
	if output.String() != "Content-Length: 38\r\n\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":null}" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestServeWritesMultipleHandlerResponses(t *testing.T) {
	request := []byte(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{}}`)
	input := bytes.NewBufferString("Content-Length: " + strconv.Itoa(len(request)) + "\r\n\r\n" + string(request))
	var output bytes.Buffer
	err := serve(input, &output, staticHandler{responses: [][]byte{
		[]byte(`{"jsonrpc":"2.0","method":"one"}`),
		[]byte(`{"jsonrpc":"2.0","method":"two"}`),
	}})
	if err == nil {
		t.Fatal("serve returned nil, want EOF after one message")
	}
	want := "Content-Length: 32\r\n\r\n{\"jsonrpc\":\"2.0\",\"method\":\"one\"}" +
		"Content-Length: 32\r\n\r\n{\"jsonrpc\":\"2.0\",\"method\":\"two\"}"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestServeStopsAfterExit(t *testing.T) {
	request := "Content-Length: 33\r\n\r\n" + `{"jsonrpc":"2.0","method":"exit"}`
	// The second message must stay unhandled: the client keeps stdin open.
	input := bytes.NewBufferString(request + request)
	handled := 0
	if err := serve(input, &bytes.Buffer{}, staticHandler{exited: true, handled: &handled}); err != nil {
		t.Fatalf("serve = %v, want nil after exit", err)
	}
	if handled != 1 {
		t.Fatalf("handled %d messages, want 1", handled)
	}
}

type staticHandler struct {
	responses [][]byte
	exited    bool
	handled   *int
}

func (h staticHandler) ExitCode() (int, bool) {
	return 0, h.exited
}

func (h staticHandler) Handle(jsontext.Value) [][]byte {
	if h.handled != nil {
		*h.handled++
	}
	return h.responses
}
