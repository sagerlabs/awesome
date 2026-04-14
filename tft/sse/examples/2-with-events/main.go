// Package main demonstrates SSE with custom event types and IDs.
package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sagerlabs/awesome/tft/sse"
)

func main() {
	server := sse.NewServer(
		sse.WithOnConnect(func(ch chan *sse.Event) {
			fmt.Println("New client connected")
		}),
		sse.WithOnDisconnect(func(ch chan *sse.Event) {
			fmt.Println("Client disconnected")
		}),
	)

	// Send different types of events
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		count := 0
		for range ticker.C {
			count++

			// Alternate between different event types
			switch count % 3 {
			case 1:
				// Message event
				event := &sse.Event{
					ID:    strconv.Itoa(count),
					Event: "message",
					Data:  fmt.Sprintf("This is a regular message #%d", count),
				}
				server.Publish(event)
				fmt.Printf("Published message event #%d\n", count)

			case 2:
				// Notification event
				event := &sse.Event{
					ID:    strconv.Itoa(count),
					Event: "notification",
					Data:  fmt.Sprintf("You have a new notification! #%d", count),
				}
				server.Publish(event)
				fmt.Printf("Published notification event #%d\n", count)

			case 0:
				// Alert event with custom retry
				event := &sse.Event{
					ID:    strconv.Itoa(count),
					Event: "alert",
					Data:  fmt.Sprintf("⚠️ Alert! This is important! #%d", count),
					Retry: 5000,
				}
				server.Publish(event)
				fmt.Printf("Published alert event #%d\n", count)
			}
		}
	}()

	http.Handle("/events", server)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
	<style>
		.message { padding: 10px; margin: 5px; border-radius: 5px; }
		.message-event { background: #e3f2fd; }
		.notification-event { background: #fff3e0; }
		.alert-event { background: #ffebee; color: #c62828; }
	</style>
</head>
<body>
	<h1>SSE with Custom Events</h1>
	<div id="messages"></div>
	<script>
		const eventSource = new EventSource('/events');
		const messagesDiv = document.getElementById('messages');
		
		function addMessage(text, className) {
			const div = document.createElement('div');
			div.className = 'message ' + className;
			div.textContent = text;
			messagesDiv.appendChild(div);
		}
		
		eventSource.addEventListener('message', function(event) {
			addMessage('[ID: ' + event.lastEventId + '] ' + event.data, 'message-event');
		});
		
		eventSource.addEventListener('notification', function(event) {
			addMessage('[ID: ' + event.lastEventId + '] ' + event.data, 'notification-event');
		});
		
		eventSource.addEventListener('alert', function(event) {
			addMessage('[ID: ' + event.lastEventId + '] ' + event.data, 'alert-event');
		});
		
		eventSource.onerror = function(err) {
			console.error('EventSource failed:', err);
		};
	</script>
</body>
</html>
`)
	})

	fmt.Println("Server starting on :8081...")
	fmt.Println("Visit http://localhost:8081 to see the example")
	http.ListenAndServe(":8081", nil)
}
