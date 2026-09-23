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

// ServeStdio reads newline-delimited JSON-RPC messages from r and writes
// responses to w until r is exhausted or ctx is cancelled. Nothing but MCP
// messages is ever written to w; diagnostics belong on stderr.
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
		var m Message
		if err := json.Unmarshal(line, &m); err != nil {
			if err := write(errorResponse(nil, CodeParseError, "parse error")); err != nil {
				return err
			}
			continue
		}
		if resp := srv.Handle(ctx, sess, &m); resp != nil {
			if err := write(resp); err != nil {
				return err
			}
		}
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			_ = write(errorResponse(nil, CodeInvalidRequest, "message exceeds size limit"))
		}
		return err
	}
	return nil
}
