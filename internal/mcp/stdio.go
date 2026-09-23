package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

// MaxMessageBytes bounds a single inbound JSON-RPC message on any transport.
const MaxMessageBytes = 1 << 20

// maxInFlight bounds concurrently executing stdio requests.
const maxInFlight = 8

// ServeStdio reads newline-delimited JSON-RPC messages from r and writes
// responses to w until r is exhausted or ctx is cancelled. Nothing but MCP
// messages is ever written to w; diagnostics belong on stderr.
//
// Requests run concurrently so a slow upstream call does not block others,
// and notifications/cancelled can stop one: its context is cancelled and no
// response is sent for it. initialize and notifications are handled in
// order, and the legacy "not initialized" gate is evaluated when a request
// is received, so lifecycle ordering is preserved.
func (srv *Server) ServeStdio(ctx context.Context, sess *Session, r io.Reader, w io.Writer) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), MaxMessageBytes)
	var wmu sync.Mutex
	write := func(m *Message) error {
		b, err := json.Marshal(m)
		if err != nil {
			return err
		}
		wmu.Lock()
		defer wmu.Unlock()
		_, err = w.Write(append(b, '\n'))
		return err
	}

	var (
		mu       sync.Mutex
		inflight = map[string]context.CancelFunc{}
		wg       sync.WaitGroup
		slots    = make(chan struct{}, maxInFlight)
	)
	defer wg.Wait()

	for sc.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if line[0] == '[' {
			// JSON-RPC batching was removed from MCP in 2025-06-18.
			if err := write(errorResponse(nil, CodeInvalidRequest, "batch requests are not supported")); err != nil {
				return err
			}
			continue
		}
		m, bad := DecodeMessage(line)
		if bad != nil {
			if err := write(bad); err != nil {
				return err
			}
			continue
		}
		if m.IsNotification() {
			if m.Method == "notifications/cancelled" {
				var p struct {
					RequestID json.RawMessage `json:"requestId"`
				}
				if json.Unmarshal(m.Params, &p) == nil {
					mu.Lock()
					if cancel, ok := inflight[string(p.RequestID)]; ok {
						cancel()
						delete(inflight, string(p.RequestID))
					}
					mu.Unlock()
				}
			}
			continue
		}
		if m.IsResponse() {
			continue
		}
		_, modern, _ := requestMeta(m.Params)
		if m.Method == "initialize" || (!modern && needsInit(m.Method) && !sess.Initialized()) {
			// Synchronous: initialize must complete before later requests,
			// and a pre-initialize rejection must reflect arrival order.
			if err := write(srv.Handle(ctx, sess, m)); err != nil {
				return err
			}
			continue
		}
		key := string(m.ID)
		rctx, cancel := context.WithCancel(ctx)
		mu.Lock()
		inflight[key] = cancel
		mu.Unlock()
		slots <- struct{}{}
		wg.Add(1)
		go func(m *Message) {
			defer wg.Done()
			defer func() { <-slots }()
			resp := srv.Handle(rctx, sess, m)
			mu.Lock()
			_, live := inflight[key]
			delete(inflight, key)
			mu.Unlock()
			cancel()
			if live && resp != nil {
				_ = write(resp)
			}
		}(m)
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			_ = write(errorResponse(nil, CodeInvalidRequest, "message exceeds size limit"))
		}
		return err
	}
	return nil
}
