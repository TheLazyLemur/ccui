package anthropic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamEvent is a parsed SSE event from the Anthropic API
type StreamEvent struct {
	Type string

	// Populated based on Type
	MessageStart      *MessageStartEvent
	ContentBlockStart *ContentBlockStartEvent
	ContentBlockDelta *ContentBlockDeltaEvent
	ContentBlockStop  *ContentBlockStopEvent
	MessageDelta      *MessageDeltaEvent
	MessageStop       *MessageStopEvent
	Ping              *PingEvent
	Error             *ErrorEvent
}

// StreamReader parses SSE events from an HTTP response body
type StreamReader struct {
	reader io.ReadCloser
	scan   *bufio.Scanner
	closed bool
}

// NewStreamReader creates a new StreamReader from an HTTP response body
func NewStreamReader(body io.ReadCloser) *StreamReader {
	return &StreamReader{
		reader: body,
		scan:   bufio.NewScanner(body),
	}
}

// Next returns the next SSE event, or io.EOF when stream ends
func (s *StreamReader) Next() (StreamEvent, error) {
	if s.closed {
		return StreamEvent{}, io.EOF
	}

	var eventType string
	var dataLine string

	// Read until we have both event and data lines
	for s.scan.Scan() {
		line := s.scan.Text()

		// Skip empty lines (event separator)
		if line == "" {
			// If we have both event and data, process the event
			if eventType != "" && dataLine != "" {
				return s.parseEvent(eventType, dataLine)
			}
			continue
		}

		// Parse event: line
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		// Parse data: line
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			continue
		}
	}

	if err := s.scan.Err(); err != nil {
		return StreamEvent{}, err
	}

	// End of stream
	return StreamEvent{}, io.EOF
}

// parseEvent parses the event type and JSON data into a StreamEvent
func (s *StreamReader) parseEvent(eventType, data string) (StreamEvent, error) {
	ev := StreamEvent{Type: eventType}

	switch eventType {
	case EventMessageStart:
		var msg MessageStartEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse message_start: %w", err)
		}
		ev.MessageStart = &msg

	case EventContentBlockStart:
		var msg ContentBlockStartEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse content_block_start: %w", err)
		}
		ev.ContentBlockStart = &msg

	case EventContentBlockDelta:
		var msg ContentBlockDeltaEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse content_block_delta: %w", err)
		}
		ev.ContentBlockDelta = &msg

	case EventContentBlockStop:
		var msg ContentBlockStopEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse content_block_stop: %w", err)
		}
		ev.ContentBlockStop = &msg

	case EventMessageDelta:
		var msg MessageDeltaEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse message_delta: %w", err)
		}
		ev.MessageDelta = &msg

	case EventMessageStop:
		var msg MessageStopEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse message_stop: %w", err)
		}
		ev.MessageStop = &msg

	case EventPing:
		var msg PingEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse ping: %w", err)
		}
		ev.Ping = &msg

	case EventError:
		var msg ErrorEvent
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return ev, fmt.Errorf("parse error: %w", err)
		}
		ev.Error = &msg
	}

	return ev, nil
}

// Close closes the underlying reader
func (s *StreamReader) Close() error {
	s.closed = true
	return s.reader.Close()
}
