package sse

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEvent_String(t *testing.T) {
	tests := []struct {
		name  string
		event *Event
		want  string
	}{
		{
			name: "basic event",
			event: &Event{
				Data: "hello world",
			},
			want: "data: hello world\n\n",
		},
		{
			name: "full event",
			event: &Event{
				ID:    "123",
				Event: "message",
				Data:  "hello world",
				Retry: 5000,
			},
			want: "id: 123\nevent: message\nretry: 5000\ndata: hello world\n\n",
		},
		{
			name: "multi-line data",
			event: &Event{
				Data: "line 1\nline 2\nline 3",
			},
			want: "data: line 1\ndata: line 2\ndata: line 3\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.String(); got != tt.want {
				t.Errorf("Event.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvent_MarshalData(t *testing.T) {
	event := &Event{}
	data := map[string]string{"key": "value"}

	err := event.MarshalData(data)
	if err != nil {
		t.Fatalf("MarshalData failed: %v", err)
	}

	var result map[string]string
	err = json.Unmarshal([]byte(event.Data), &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Expected key=value, got key=%s", result["key"])
	}
}

func TestEvent_UnmarshalData(t *testing.T) {
	event := &Event{
		Data: `{"key":"value"}`,
	}

	var result map[string]string
	err := event.UnmarshalData(&result)
	if err != nil {
		t.Fatalf("UnmarshalData failed: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Expected key=value, got key=%s", result["key"])
	}
}

func TestServer_SubscribeUnsubscribe(t *testing.T) {
	server := NewServer()

	ch := server.Subscribe()
	if server.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", server.ClientCount())
	}

	server.Unsubscribe(ch)
	if server.ClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", server.ClientCount())
	}
}

func TestServer_Publish(t *testing.T) {
	server := NewServer()

	ch1 := server.Subscribe()
	ch2 := server.Subscribe()

	event := &Event{Data: "test"}
	server.Publish(event)

	select {
	case e := <-ch1:
		if e.Data != "test" {
			t.Errorf("Expected data=test, got data=%s", e.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for event on ch1")
	}

	select {
	case e := <-ch2:
		if e.Data != "test" {
			t.Errorf("Expected data=test, got data=%s", e.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for event on ch2")
	}
}

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want *Event
	}{
		{
			name: "basic event",
			text: "data: hello",
			want: &Event{Data: "hello"},
		},
		{
			name: "full event",
			text: "id: 123\nevent: message\ndata: hello\nretry: 5000",
			want: &Event{ID: "123", Event: "message", Data: "hello", Retry: 5000},
		},
		{
			name: "multi-line data",
			text: "data: line1\ndata: line2",
			want: &Event{Data: "line1\nline2"},
		},
		{
			name: "with comments",
			text: ": this is a comment\ndata: hello",
			want: &Event{Data: "hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEvent(tt.text, nil)
			if err != nil {
				t.Fatalf("parseEvent failed: %v", err)
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID: got %q, want %q", got.ID, tt.want.ID)
			}
			if got.Event != tt.want.Event {
				t.Errorf("Event: got %q, want %q", got.Event, tt.want.Event)
			}
			if got.Data != tt.want.Data {
				t.Errorf("Data: got %q, want %q", got.Data, tt.want.Data)
			}
			if got.Retry != tt.want.Retry {
				t.Errorf("Retry: got %d, want %d", got.Retry, tt.want.Retry)
			}
		})
	}
}

func TestEventReader_Read(t *testing.T) {
	data := "data: event1\n\ndata: event2\n\n"
	r := NewEventReader(io.NopCloser(strings.NewReader(data)), nil)
	defer r.Close()

	event1, err := r.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if event1.Data != "event1" {
		t.Errorf("Expected event1, got %s", event1.Data)
	}

	event2, err := r.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if event2.Data != "event2" {
		t.Errorf("Expected event2, got %s", event2.Data)
	}

	_, err = r.Read()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestServer_WithOptions(t *testing.T) {
	connectCalled := false
	disconnectCalled := false

	server := NewServer(
		WithBufferSize(200),
		WithOnConnect(func(ch chan *Event) {
			connectCalled = true
		}),
		WithOnDisconnect(func(ch chan *Event) {
			disconnectCalled = true
		}),
	)

	ch := server.Subscribe()
	if !connectCalled {
		t.Error("Expected OnConnect to be called")
	}

	server.Unsubscribe(ch)
	if !disconnectCalled {
		t.Error("Expected OnDisconnect to be called")
	}
}

func TestClient_WithOptions(t *testing.T) {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	client := NewClient(
		"http://example.com",
		WithHTTPClient(httpClient),
		WithLastEventID("last-id"),
		WithRetry(10000),
	)

	if client.HTTPClient != httpClient {
		t.Error("Expected custom HTTP client")
	}
	if client.LastEventID != "last-id" {
		t.Error("Expected LastEventID to be set")
	}
	if client.Retry != 10000 {
		t.Error("Expected Retry to be set")
	}
}
