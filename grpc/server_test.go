package grpc

import (
	"context"
	"testing"
)

func TestServerHandle(t *testing.T) {
	server := NewServer()

	server.Handle("TestAction", func(ctx context.Context, clientID string, payload []byte) ([]byte, error) {
		return []byte("response"), nil
	})

	handler, ok := server.handlers.Get("TestAction")
	if !ok {
		t.Error("handler not found")
	}

	result, err := handler(nil, "client1", []byte("request"))
	if err != nil {
		t.Error(err)
	}

	if string(result) != "response" {
		t.Errorf("expected 'response', got '%s'", result)
	}
}
