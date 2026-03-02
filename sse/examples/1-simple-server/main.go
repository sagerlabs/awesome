// Package main is a simple SSE server example that sends periodic messages.
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sagerlabs/awesome/sse"
)

func main() {
	// Create a new SSE server
	server := sse.NewServer()

	// Start a goroutine to send periodic messages
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		count := 0
		for range ticker.C {
			count++
			event := &sse.Event{
				Data: fmt.Sprintf("Hello, SSE! Message #%d", count),
			}
			server.Publish(event)
			fmt.Printf("Published message #%d\n", count)
		}
	}()

	// Set up HTTP routes
	http.Handle("/events", server)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<body>
	<h1>SSE Simple Example</h1>
	<div id="messages"></div>
	<script>
		const eventSource = new EventSource('/events');
		const messagesDiv = document.getElementById('messages');
		
		eventSource.onmessage = function(event) {
			const p = document.createElement('p');
			p.textContent = event.data;
			messagesDiv.appendChild(p);
		};
		
		eventSource.onerror = function(err) {
			console.error('EventSource failed:', err);
		};
	</script>
</body>
</html>
`)
	})

	fmt.Println("Server starting on :8080...")
	fmt.Println("Visit http://localhost:8080 to see the example")
	http.ListenAndServe(":8080", nil)
}
