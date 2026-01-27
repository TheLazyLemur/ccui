package acp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Transport handles JSON-RPC communication
type Transport interface {
	// Send sends a request and blocks for response
	Send(method string, params any) (json.RawMessage, error)

	// Notify sends a notification (no response expected)
	Notify(method string, params any)

	// Respond sends a response to an incoming request
	Respond(id *int, result json.RawMessage)

	// OnMethod registers a handler for incoming methods (notifications)
	OnMethod(handler func(method string, params json.RawMessage, id *int))

	// Close shuts down the transport
	Close() error
}

// StdioTransport implements Transport over stdin/stdout pipes
type StdioTransport struct {
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	callbacks map[int]chan json.RawMessage
	errors    map[int]chan *RPCError
	msgID     int
	mu        sync.Mutex
	handler   func(method string, params json.RawMessage, id *int)
	done      chan struct{}
	closeOnce sync.Once
}

// NewStdioTransport creates a new transport
func NewStdioTransport(stdin io.WriteCloser, stdout io.Reader) *StdioTransport {
	t := &StdioTransport{
		stdin:     stdin,
		stdout:    bufio.NewScanner(stdout),
		callbacks: make(map[int]chan json.RawMessage),
		errors:    make(map[int]chan *RPCError),
		done:      make(chan struct{}),
	}
	go t.readLoop()
	return t
}

func (t *StdioTransport) readLoop() {
	for t.stdout.Scan() {
		line := t.stdout.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		// Check Method BEFORE ID - requests have both
		if msg.Method != "" {
			if t.handler != nil {
				t.handler(msg.Method, msg.Params, msg.ID)
			}
		} else if msg.ID != nil {
			t.mu.Lock()
			if ch, ok := t.callbacks[*msg.ID]; ok {
				if msg.Error != nil {
					if errCh, ok := t.errors[*msg.ID]; ok {
						errCh <- msg.Error
					}
				}
				ch <- msg.Result
				delete(t.callbacks, *msg.ID)
				delete(t.errors, *msg.ID)
			}
			t.mu.Unlock()
		}
	}
}

// Send sends a request and blocks for response
func (t *StdioTransport) Send(method string, params any) (json.RawMessage, error) {
	t.mu.Lock()
	t.msgID++
	id := t.msgID
	ch := make(chan json.RawMessage, 1)
	errCh := make(chan *RPCError, 1)
	t.callbacks[id] = ch
	t.errors[id] = errCh
	t.mu.Unlock()

	paramsJSON, _ := json.Marshal(params)
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  paramsJSON,
	}

	data, _ := json.Marshal(msg)
	if _, err := t.stdin.Write(append(data, '\n')); err != nil {
		t.mu.Lock()
		delete(t.callbacks, id)
		delete(t.errors, id)
		t.mu.Unlock()
		return nil, err
	}

	select {
	case result := <-ch:
		select {
		case rpcErr := <-errCh:
			if rpcErr != nil {
				return nil, fmt.Errorf("rpc error %d: %s", rpcErr.Code, rpcErr.Message)
			}
		default:
		}
		return result, nil
	case <-t.done:
		return nil, fmt.Errorf("connection closed")
	}
}

// Notify sends a notification (no response expected)
func (t *StdioTransport) Notify(method string, params any) {
	paramsJSON, _ := json.Marshal(params)
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}
	data, _ := json.Marshal(msg)
	t.stdin.Write(append(data, '\n'))
}

// Respond sends a response to an incoming request
func (t *StdioTransport) Respond(id *int, result json.RawMessage) {
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(msg)
	t.stdin.Write(append(data, '\n'))
}

// OnMethod registers a handler for incoming method calls
func (t *StdioTransport) OnMethod(handler func(method string, params json.RawMessage, id *int)) {
	t.handler = handler
}

// Close shuts down the transport
func (t *StdioTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)
	})
	return t.stdin.Close()
}
