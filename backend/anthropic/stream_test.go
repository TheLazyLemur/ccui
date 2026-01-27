package anthropic

import (
	"io"
	"strings"
	"testing"
)

func TestStreamReader_TextResponse(t *testing.T) {
	// SSE for simple text response
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	// collect events
	var events []StreamEvent
	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, ev)
	}

	if len(events) != 7 {
		t.Fatalf("expected 7 events, got %d", len(events))
	}

	// message_start
	if events[0].Type != EventMessageStart {
		t.Errorf("expected message_start, got %s", events[0].Type)
	}
	if events[0].MessageStart == nil || events[0].MessageStart.Message.ID != "msg_123" {
		t.Errorf("message_start missing message data")
	}

	// content_block_start
	if events[1].Type != EventContentBlockStart {
		t.Errorf("expected content_block_start, got %s", events[1].Type)
	}
	if events[1].ContentBlockStart == nil || events[1].ContentBlockStart.ContentBlock.Type != "text" {
		t.Errorf("content_block_start missing block data")
	}

	// content_block_delta (Hello)
	if events[2].Type != EventContentBlockDelta {
		t.Errorf("expected content_block_delta, got %s", events[2].Type)
	}
	if events[2].ContentBlockDelta == nil || events[2].ContentBlockDelta.Delta.Text != "Hello" {
		t.Errorf("expected 'Hello', got %q", events[2].ContentBlockDelta.Delta.Text)
	}

	// content_block_delta ( world)
	if events[3].ContentBlockDelta == nil || events[3].ContentBlockDelta.Delta.Text != " world" {
		t.Errorf("expected ' world', got %q", events[3].ContentBlockDelta.Delta.Text)
	}

	// content_block_stop
	if events[4].Type != EventContentBlockStop {
		t.Errorf("expected content_block_stop, got %s", events[4].Type)
	}

	// message_delta
	if events[5].Type != EventMessageDelta {
		t.Errorf("expected message_delta, got %s", events[5].Type)
	}
	if events[5].MessageDelta == nil || events[5].MessageDelta.Delta.StopReason != "end_turn" {
		t.Errorf("message_delta missing stop_reason")
	}

	// message_stop
	if events[6].Type != EventMessageStop {
		t.Errorf("expected message_stop, got %s", events[6].Type)
	}
}

func TestStreamReader_ToolUse(t *testing.T) {
	// SSE for tool_use response
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_456","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":50,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_abc","name":"get_weather","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"loc"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"ation\": \"NYC\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":25}}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	var events []StreamEvent
	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, ev)
	}

	// content_block_start should have tool_use
	if events[1].ContentBlockStart == nil {
		t.Fatal("missing content_block_start")
	}
	block := events[1].ContentBlockStart.ContentBlock
	if block.Type != "tool_use" || block.ID != "toolu_abc" || block.Name != "get_weather" {
		t.Errorf("tool_use block mismatch: %+v", block)
	}

	// deltas should have partial_json
	if events[2].ContentBlockDelta.Delta.PartialJSON != "{\"loc" {
		t.Errorf("expected partial json, got %q", events[2].ContentBlockDelta.Delta.PartialJSON)
	}

	// message_delta stop_reason should be tool_use
	if events[5].MessageDelta.Delta.StopReason != "tool_use" {
		t.Errorf("expected tool_use stop reason")
	}
}

func TestStreamReader_Error(t *testing.T) {
	sseData := `event: error
data: {"type":"error","error":{"type":"overloaded_error","message":"API is overloaded"}}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	ev, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.Type != EventError {
		t.Errorf("expected error event, got %s", ev.Type)
	}
	if ev.Error == nil || ev.Error.Error.Type != "overloaded_error" {
		t.Errorf("error event mismatch: %+v", ev.Error)
	}
}

func TestStreamReader_Ping(t *testing.T) {
	sseData := `event: ping
data: {"type":"ping"}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	ev, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != EventPing {
		t.Errorf("expected ping, got %s", ev.Type)
	}

	ev, err = reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != EventMessageStop {
		t.Errorf("expected message_stop, got %s", ev.Type)
	}
}

func TestStreamReader_Thinking(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_789","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me think..."}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig123"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	var events []StreamEvent
	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, ev)
	}

	// thinking block
	if events[1].ContentBlockStart.ContentBlock.Type != "thinking" {
		t.Errorf("expected thinking block")
	}

	// thinking delta
	if events[2].ContentBlockDelta.Delta.Thinking != "Let me think..." {
		t.Errorf("expected thinking text")
	}

	// signature delta
	if events[3].ContentBlockDelta.Delta.Signature != "sig123" {
		t.Errorf("expected signature")
	}
}

func TestStreamReader_InvalidJSON(t *testing.T) {
	sseData := `event: message_start
data: {invalid json}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	_, err := reader.Next()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestStreamReader_EmptyLines(t *testing.T) {
	// Extra blank lines should be ignored
	sseData := `

event: ping
data: {"type":"ping"}


event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	ev, err := reader.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != EventPing {
		t.Errorf("expected ping, got %s", ev.Type)
	}
}

func TestStreamReader_Close(t *testing.T) {
	sseData := `event: ping
data: {"type":"ping"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))
	reader.Close()

	// After close, Next should return EOF
	_, err := reader.Next()
	if err != io.EOF {
		t.Errorf("expected EOF after close, got %v", err)
	}
}

func TestCollectTextDeltas(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":10,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	var textBuilder strings.Builder
	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Type == EventContentBlockDelta && ev.ContentBlockDelta != nil {
			if ev.ContentBlockDelta.Delta.Type == DeltaTypeText {
				textBuilder.WriteString(ev.ContentBlockDelta.Delta.Text)
			}
		}
	}

	if textBuilder.String() != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", textBuilder.String())
	}
}

func TestCollectToolInput(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_456","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null,"usage":{"input_tokens":50,"output_tokens":1}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_abc","name":"get_weather","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"loc"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"ation\":"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":" \"NYC\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_stop
data: {"type":"message_stop"}

`

	reader := NewStreamReader(io.NopCloser(strings.NewReader(sseData)))

	var jsonBuilder strings.Builder
	for {
		ev, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Type == EventContentBlockDelta && ev.ContentBlockDelta != nil {
			if ev.ContentBlockDelta.Delta.Type == DeltaTypeInputJSON {
				jsonBuilder.WriteString(ev.ContentBlockDelta.Delta.PartialJSON)
			}
		}
	}

	expected := `{"location": "NYC"}`
	if jsonBuilder.String() != expected {
		t.Errorf("expected %q, got %q", expected, jsonBuilder.String())
	}
}
