package acp

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

func TestTransport_SendReceive(t *testing.T) {
	// given: a transport with simulated stdin/stdout
	// io.Pipe() returns (reader, writer)
	serverReader, clientWriter := io.Pipe() // client writes to server
	clientReader, serverWriter := io.Pipe() // server writes to client

	transport := NewStdioTransport(clientWriter, clientReader)
	defer transport.Close()

	// Simulate server: read request, send response
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		n, err := serverReader.Read(buf)
		if err != nil {
			t.Errorf("server read: %v", err)
			return
		}
		var req JSONRPCMessage
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			t.Errorf("unmarshal request: %v", err)
			return
		}

		// Verify request
		if req.Method != "test/echo" {
			t.Errorf("expected method test/echo, got %s", req.Method)
		}
		if req.ID == nil {
			t.Errorf("expected ID in request")
			return
		}

		// Send response
		resp := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"echoed": true}`),
		}
		respData, _ := json.Marshal(resp)
		serverWriter.Write(append(respData, '\n'))
	}()

	// when: sending a request
	result, err := transport.Send("test/echo", map[string]string{"msg": "hello"})

	// then: should receive response
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	var resultData struct {
		Echoed bool `json:"echoed"`
	}
	if err := json.Unmarshal(result, &resultData); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !resultData.Echoed {
		t.Error("expected echoed=true")
	}

	serverReader.Close()
	serverWriter.Close()
	wg.Wait()
}

func TestTransport_CallbackRouting(t *testing.T) {
	// given: a transport with simulated stdin/stdout
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	transport := NewStdioTransport(clientWriter, clientReader)
	defer transport.Close()

	// Simulate server: read 2 requests, respond out of order
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var requests []JSONRPCMessage
		buf := make([]byte, 4096)

		// Read first request
		n, _ := serverReader.Read(buf)
		var req1 JSONRPCMessage
		json.Unmarshal(buf[:n], &req1)
		requests = append(requests, req1)

		// Read second request
		n, _ = serverReader.Read(buf)
		var req2 JSONRPCMessage
		json.Unmarshal(buf[:n], &req2)
		requests = append(requests, req2)

		// Respond in reverse order to test routing
		// Each response includes the method name so we verify correct routing
		resp2 := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      requests[1].ID,
			Result:  json.RawMessage(fmt.Sprintf(`{"method": %q}`, requests[1].Method)),
		}
		data2, _ := json.Marshal(resp2)
		serverWriter.Write(append(data2, '\n'))

		resp1 := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      requests[0].ID,
			Result:  json.RawMessage(fmt.Sprintf(`{"method": %q}`, requests[0].Method)),
		}
		data1, _ := json.Marshal(resp1)
		serverWriter.Write(append(data1, '\n'))
	}()

	// when: sending 2 requests concurrently with distinct methods
	var results [2]json.RawMessage
	var errs [2]error
	var clientWg sync.WaitGroup
	clientWg.Add(2)

	go func() {
		defer clientWg.Done()
		results[0], errs[0] = transport.Send("method_A", nil)
	}()
	go func() {
		defer clientWg.Done()
		results[1], errs[1] = transport.Send("method_B", nil)
	}()

	clientWg.Wait()

	// then: each request gets response with matching method (proves routing by ID)
	for i, err := range errs {
		if err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}

	var r1, r2 struct{ Method string }
	json.Unmarshal(results[0], &r1)
	json.Unmarshal(results[1], &r2)

	// Results should match the method each goroutine sent
	if r1.Method != "method_A" {
		t.Errorf("method_A call got response %q", r1.Method)
	}
	if r2.Method != "method_B" {
		t.Errorf("method_B call got response %q", r2.Method)
	}

	serverReader.Close()
	serverWriter.Close()
	wg.Wait()
}

func TestTransport_MethodHandler(t *testing.T) {
	// given: a transport with method handler
	_, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	transport := NewStdioTransport(clientWriter, clientReader)
	defer transport.Close()

	received := make(chan struct {
		method string
		params json.RawMessage
		id     *int
	}, 1)

	transport.OnMethod(func(method string, params json.RawMessage, id *int) {
		received <- struct {
			method string
			params json.RawMessage
			id     *int
		}{method, params, id}
	})

	// when: server sends a method call (notification)
	notification := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "session/update",
		Params:  json.RawMessage(`{"sessionId": "abc"}`),
	}
	data, _ := json.Marshal(notification)
	serverWriter.Write(append(data, '\n'))

	// then: handler should be called
	select {
	case r := <-received:
		if r.method != "session/update" {
			t.Errorf("got method %q, want session/update", r.method)
		}
		if r.id != nil {
			t.Error("expected nil id for notification")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for method handler")
	}

	serverWriter.Close()
}

func TestTransport_Notify(t *testing.T) {
	// given: a transport
	serverReader, clientWriter := io.Pipe()
	clientReader, _ := io.Pipe()

	transport := NewStdioTransport(clientWriter, clientReader)
	defer transport.Close()

	received := make(chan JSONRPCMessage, 1)
	go func() {
		buf := make([]byte, 4096)
		n, _ := serverReader.Read(buf)
		var msg JSONRPCMessage
		json.Unmarshal(buf[:n], &msg)
		received <- msg
	}()

	// when: sending a notification
	transport.Notify("session/cancel", map[string]string{"sessionId": "abc"})

	// then: server should receive notification without ID
	select {
	case msg := <-received:
		if msg.Method != "session/cancel" {
			t.Errorf("got method %q, want session/cancel", msg.Method)
		}
		if msg.ID != nil {
			t.Error("notification should not have ID")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notification")
	}

	serverReader.Close()
}

func TestTransport_ErrorResponse(t *testing.T) {
	// given: a transport
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	transport := NewStdioTransport(clientWriter, clientReader)
	defer transport.Close()

	go func() {
		buf := make([]byte, 4096)
		n, _ := serverReader.Read(buf)
		var req JSONRPCMessage
		json.Unmarshal(buf[:n], &req)

		// Send error response
		resp := JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32600, Message: "Invalid Request"},
		}
		data, _ := json.Marshal(resp)
		serverWriter.Write(append(data, '\n'))
	}()

	// when: sending a request that gets an error
	_, err := transport.Send("bad/method", nil)

	// then: should return error
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "rpc error -32600: Invalid Request" {
		t.Errorf("got error %q", err)
	}

	serverReader.Close()
	serverWriter.Close()
}
