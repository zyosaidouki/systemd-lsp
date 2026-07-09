package main

import (
	"bufio"
	"bytes"
	"encoding/json"
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

type staticHandler struct {
	responses [][]byte
}

func (h staticHandler) Handle(json.RawMessage) [][]byte {
	return h.responses
}
