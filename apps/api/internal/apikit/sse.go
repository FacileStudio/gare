package apikit

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Event is one server-sent event. ID and Name are left out of the frame when empty, and Data may
// span lines, each sent as a data field of its own.
type Event struct {
	ID   string
	Name string
	Data string
}

// Stream writes server-sent events and flushes each one, so a browser sees a line the moment the
// daemon has it rather than when a buffer fills.
type Stream struct {
	w  http.ResponseWriter
	rc *http.ResponseController
	mu sync.Mutex
}

// OpenStream answers 200 as text/event-stream and flushes the headers, so the client's EventSource
// opens before the first event. It fails when the connection cannot flush, since a stream that
// only arrives once the request ends is no stream.
func OpenStream(w http.ResponseWriter) (*Stream, error) {
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	s := &Stream{w: w, rc: http.NewResponseController(w)}
	return s, s.rc.Flush()
}

// Send writes one event and flushes it. It is safe to call from several goroutines.
func (s *Stream) Send(ev Event) error {
	var frame strings.Builder
	if ev.ID != "" {
		fmt.Fprintf(&frame, "id: %s\n", ev.ID)
	}
	if ev.Name != "" {
		fmt.Fprintf(&frame, "event: %s\n", ev.Name)
	}
	for _, line := range strings.Split(ev.Data, "\n") {
		fmt.Fprintf(&frame, "data: %s\n", strings.TrimSuffix(line, "\r"))
	}
	frame.WriteString("\n")
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.w.Write([]byte(frame.String())); err != nil {
		return err
	}
	return s.rc.Flush()
}

// LineWriter turns a byte stream, such as journalctl's output, into one event per line. A line the
// writer has not finished stays buffered until its newline arrives.
type LineWriter struct {
	stream  *Stream
	partial string
}

// NewLineWriter returns a writer that sends each complete line to stream as a data event.
func NewLineWriter(stream *Stream) *LineWriter {
	return &LineWriter{stream: stream}
}

// Write sends every line p completes.
func (l *LineWriter) Write(p []byte) (int, error) {
	l.partial += string(p)
	for {
		line, rest, found := strings.Cut(l.partial, "\n")
		if !found {
			return len(p), nil
		}
		l.partial = rest
		if err := l.stream.Send(Event{Data: line}); err != nil {
			return 0, err
		}
	}
}
