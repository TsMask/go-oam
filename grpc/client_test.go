package grpc

import (
	"context"
	"testing"
)

func TestClientHandle(t *testing.T) {
	client := NewClient("localhost:50051")
	client.Handle("TestAction", func(ctx context.Context, payload []byte) ([]byte, error) {
		return []byte("response"), nil
	})

	handler, ok := client.handlers.Get("TestAction")
	if !ok {
		t.Error("handler not found")
	}

	result, err := handler(nil, []byte("request"))
	if err != nil {
		t.Error(err)
	}

	if string(result) != "response" {
		t.Errorf("expected 'response', got '%s'", result)
	}
}
