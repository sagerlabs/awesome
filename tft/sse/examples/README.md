# SSE Examples

This directory contains examples of using the SSE (Server-Sent Events) package from simple to complex.

## Examples

### 1. Simple Server

A basic SSE server that sends periodic messages.

**Features:**
- Basic SSE server setup
- Periodic message broadcasting
- Simple HTML client

**To run:**
```bash
cd 1-simple-server
go run main.go
```

Then visit http://localhost:8080

### 2. With Custom Events

Demonstrates SSE with custom event types and IDs.

**Features:**
- Multiple event types (message, notification, alert)
- Event IDs for tracking
- Connect/disconnect callbacks
- Custom retry intervals
- Styled HTML client

**To run:**
```bash
cd 2-with-events
go run main.go
```

Then visit http://localhost:8081

### 3. JSON Data

Sends structured JSON data over SSE.

**Features:**
- JSON marshaling/unmarshaling
- Multiple data types (stocks, weather)
- Real-time updates
- Rich HTML client with styling

**To run:**
```bash
cd 3-json-data
go run main.go
```

Then visit http://localhost:8082

### 4. Client Example

Demonstrates how to use the SSE client.

**Features:**
- Basic client connection
- Streaming with automatic reconnection
- Custom client options
- Background test server

**To run:**
```bash
cd 4-client-example
go run main.go
```

## Key Concepts

### Server Setup

```go
server := sse.NewServer()
http.Handle("/events", server)
```

### Publishing Events

```go
event := &sse.Event{
    ID:    "123",
    Event: "custom-event",
    Data:  "Hello, World!",
    Retry: 5000,
}
server.Publish(event)
```

### JSON Data

```go
event := &sse.Event{}
event.MarshalData(myStruct)
server.Publish(event)
```

### Client Usage

```go
client := sse.NewClient("http://example.com/events")
reader, err := client.Connect(ctx)
event, err := reader.Read()
```

### Streaming Client

```go
eventCh, errCh := client.Stream(ctx)
for event := range eventCh {
    // handle event
}
```

## Browser Client

```javascript
const eventSource = new EventSource('/events');

eventSource.onmessage = function(event) {
    console.log(event.data);
};

eventSource.addEventListener('custom-event', function(event) {
    console.log('Custom event:', event.data);
});
```
