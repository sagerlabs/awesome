// Package sse provides a robust Server-Sent Events (SSE) implementation for Go.
package sse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultRetry is the default retry interval in milliseconds
	DefaultRetry = 3000
	// ContentType is the SSE content type
	ContentType = "text/event-stream"
	// CacheControl is the recommended cache control header
	CacheControl = "no-cache"
	// ConnectionHeader is the recommended connection header
	ConnectionHeader = "keep-alive"
)

// Event represents a single SSE event
type Event struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// String returns the string representation of the event in SSE format
func (e *Event) String() string {
	var buf bytes.Buffer

	if e.ID != "" {
		buf.WriteString("id: " + e.ID + "\n")
	}
	if e.Event != "" {
		buf.WriteString("event: " + e.Event + "\n")
	}
	if e.Retry > 0 {
		buf.WriteString(fmt.Sprintf("retry: %d\n", e.Retry))
	}

	lines := strings.Split(e.Data, "\n")
	for _, line := range lines {
		buf.WriteString("data: " + line + "\n")
	}

	buf.WriteString("\n")
	return buf.String()
}

// MarshalJSON marshals the event data to JSON
func (e *Event) MarshalJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	e.Data = string(data)
	return nil
}

// UnmarshalJSON unmarshals the event data from JSON
func (e *Event) UnmarshalJSON(v interface{}) error {
	return json.Unmarshal([]byte(e.Data), v)
}

// Server represents an SSE server that can send events to multiple clients
type Server struct {
	mu           sync.RWMutex
	clients      map[chan *Event]struct{}
	onConnect    func(chan *Event)
	onDisconnect func(chan *Event)
	bufferSize   int
}

// ServerOption is a function that configures a Server
type ServerOption func(*Server)

// WithBufferSize sets the buffer size for client channels
func WithBufferSize(size int) ServerOption {
	return func(s *Server) {
		s.bufferSize = size
	}
}

// WithOnConnect sets the callback for when a client connects
func WithOnConnect(fn func(chan *Event)) ServerOption {
	return func(s *Server) {
		s.onConnect = fn
	}
}

// WithOnDisconnect sets the callback for when a client disconnects
func WithOnDisconnect(fn func(chan *Event)) ServerOption {
	return func(s *Server) {
		s.onDisconnect = fn
	}
}

// NewServer creates a new SSE server
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		clients:    make(map[chan *Event]struct{}),
		bufferSize: 100,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Subscribe creates a new client subscription
func (s *Server) Subscribe() chan *Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *Event, s.bufferSize)
	s.clients[ch] = struct{}{}

	if s.onConnect != nil {
		s.onConnect(ch)
	}

	return ch
}

// Unsubscribe removes a client subscription
func (s *Server) Unsubscribe(ch chan *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[ch]; ok {
		delete(s.clients, ch)
		close(ch)

		if s.onDisconnect != nil {
			s.onDisconnect(ch)
		}
	}
}

// Publish sends an event to all connected clients
func (s *Server) Publish(event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for ch := range s.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

// ClientCount returns the number of connected clients
func (s *Server) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// ServeHTTP implements http.Handler for SSE streaming
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", ContentType)
	w.Header().Set("Cache-Control", CacheControl)
	w.Header().Set("Connection", ConnectionHeader)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := s.Subscribe()
	defer s.Unsubscribe(ch)

	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case event := <-ch:
			if event == nil {
				return
			}
			fmt.Fprint(w, event.String())
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// Client represents an SSE client that can receive events from a server
type Client struct {
	URL         string
	HTTPClient  *http.Client
	LastEventID string
	Retry       int
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithHTTPClient sets the HTTP client for the SSE client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.HTTPClient = client
	}
}

// WithLastEventID sets the last event ID for reconnection
func WithLastEventID(id string) ClientOption {
	return func(c *Client) {
		c.LastEventID = id
	}
}

// WithRetry sets the retry interval in milliseconds
func WithRetry(retry int) ClientOption {
	return func(c *Client) {
		c.Retry = retry
	}
}

// NewClient creates a new SSE client
func NewClient(url string, opts ...ClientOption) *Client {
	c := &Client{
		URL:        url,
		HTTPClient: &http.Client{},
		Retry:      DefaultRetry,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Connect connects to the SSE server and returns an EventReader
func (c *Client) Connect(ctx context.Context) (*EventReader, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", ContentType)
	req.Header.Set("Cache-Control", CacheControl)
	if c.LastEventID != "" {
		req.Header.Set("Last-Event-ID", c.LastEventID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != ContentType {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return NewEventReader(resp.Body, c), nil
}

// EventReader reads events from an SSE stream
type EventReader struct {
	scanner *bufio.Scanner
	client  *Client
	body    io.ReadCloser
}

// NewEventReader creates a new EventReader
func NewEventReader(r io.ReadCloser, client *Client) *EventReader {
	scanner := bufio.NewScanner(r)
	scanner.Split(scanSSE)
	return &EventReader{
		scanner: scanner,
		client:  client,
		body:    r,
	}
}

func scanSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}

	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		return i + 4, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

// Read reads the next event from the stream
func (r *EventReader) Read() (*Event, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	return parseEvent(r.scanner.Text(), r.client)
}

func parseEvent(text string, client *Client) (*Event, error) {
	event := &Event{}
	var dataBuf bytes.Buffer

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		colon := strings.Index(line, ":")
		var field, value string
		if colon == -1 {
			field = line
			value = ""
		} else {
			field = line[:colon]
			value = strings.TrimSpace(line[colon+1:])
		}

		switch field {
		case "id":
			event.ID = value
			if client != nil {
				client.LastEventID = value
			}
		case "event":
			event.Event = value
		case "data":
			if dataBuf.Len() > 0 {
				dataBuf.WriteString("\n")
			}
			dataBuf.WriteString(value)
		case "retry":
			if retry, err := strconv.Atoi(value); err == nil {
				event.Retry = retry
				if client != nil {
					client.Retry = retry
				}
			}
		}
	}

	event.Data = dataBuf.String()
	return event, nil
}

// Close closes the event reader
func (r *EventReader) Close() error {
	return r.body.Close()
}

// Stream streams events to a channel until context is cancelled
func (c *Client) Stream(ctx context.Context) (<-chan *Event, <-chan error) {
	eventCh := make(chan *Event)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			reader, err := c.Connect(ctx)
			if err != nil {
				errCh <- err
				time.Sleep(time.Duration(c.Retry) * time.Millisecond)
				continue
			}

			for {
				event, err := reader.Read()
				if err != nil {
					reader.Close()
					if err == io.EOF {
						time.Sleep(time.Duration(c.Retry) * time.Millisecond)
						break
					}
					errCh <- err
					time.Sleep(time.Duration(c.Retry) * time.Millisecond)
					break
				}

				select {
				case eventCh <- event:
				case <-ctx.Done():
					reader.Close()
					errCh <- ctx.Err()
					return
				}
			}
		}
	}()

	return eventCh, errCh
}
