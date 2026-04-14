// Package main demonstrates how to use the SSE client.
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sagerlabs/awesome/tft/sse"
)

func main() {
	// First, start a simple server in the background for our client to connect to
	server := sse.NewServer()
	go startBackgroundServer(server)
	go sendTestEvents(server)

	// Wait a bit for the server to start
	time.Sleep(500 * time.Millisecond)

	fmt.Println("=== SSE Client Example ===")
	fmt.Println()

	// Example 1: Basic client connection
	fmt.Println("Example 1: Basic client connection")
	fmt.Println("----------------------------------")
	basicClientExample()
	fmt.Println()

	// Example 2: Streaming client with reconnection
	fmt.Println("Example 2: Streaming client with reconnection")
	fmt.Println("---------------------------------------------")
	streamingClientExample()
	fmt.Println()

	// Example 3: Client with custom options
	fmt.Println("Example 3: Client with custom options")
	fmt.Println("------------------------------------")
	customClientExample()
	fmt.Println()

	fmt.Println("All examples completed!")
}

func startBackgroundServer(server *sse.Server) {
	http.Handle("/events", server)
	fmt.Println("Background server running on :8083")
	http.ListenAndServe(":8083", nil)
}

func sendTestEvents(server *sse.Server) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	count := 0
	for range ticker.C {
		count++
		event := &sse.Event{
			ID:    fmt.Sprintf("%d", count),
			Event: "test-event",
			Data:  fmt.Sprintf("Test message #%d", count),
		}
		server.Publish(event)
	}
}

func basicClientExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a client
	client := sse.NewClient("http://localhost:8083/events")

	// Connect to the server
	reader, err := client.Connect(ctx)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer reader.Close()

	fmt.Println("Connected to server!")

	// Read a few events
	for i := 0; i < 3; i++ {
		event, err := reader.Read()
		if err != nil {
			fmt.Printf("Error reading event: %v\n", err)
			return
		}
		fmt.Printf("Received: ID=%s, Event=%s, Data=%s\n",
			event.ID, event.Event, event.Data)
	}
}

func streamingClientExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Create a client
	client := sse.NewClient("http://localhost:8083/events")

	// Start streaming
	eventCh, errCh := client.Stream(ctx)

	fmt.Println("Streaming events...")

	eventCount := 0
	for {
		select {
		case event := <-eventCh:
			if event == nil {
				return
			}
			eventCount++
			fmt.Printf("[Stream %d] ID=%s, Data=%s\n",
				eventCount, event.ID, event.Data)
		case err := <-errCh:
			fmt.Printf("Stream error: %v\n", err)
			return
		case <-ctx.Done():
			fmt.Println("Stream context done")
			return
		}
	}
}

func customClientExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a custom HTTP client
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create SSE client with custom options
	client := sse.NewClient(
		"http://localhost:8083/events",
		sse.WithHTTPClient(httpClient),
		sse.WithLastEventID("100"),
		sse.WithRetry(2000),
	)

	fmt.Printf("Client configured with LastEventID=%s, Retry=%dms\n",
		client.LastEventID, client.Retry)

	// Connect and read one event
	reader, err := client.Connect(ctx)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer reader.Close()

	event, err := reader.Read()
	if err != nil {
		fmt.Printf("Error reading event: %v\n", err)
		return
	}

	fmt.Printf("Received event: %s\n", event.Data)
	fmt.Printf("Client's LastEventID now: %s\n", client.LastEventID)
}
