package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"c-z.dev/go-micro/api"
	"c-z.dev/go-micro/client"
	"c-z.dev/go-micro/client/selector"
	raw "c-z.dev/go-micro/codec/bytes"
	"c-z.dev/go-micro/logger"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	"github.com/gobwas/ws/wsutil"
)

// serveWebsocket will stream rpc back over websockets assuming json
func serveWebsocket(ctx context.Context, w http.ResponseWriter, r *http.Request, service *api.Service, c client.Client) {
	var op ws.OpCode

	ct := r.Header.Get("Content-Type")
	// Strip charset from Content-Type (like `application/json; charset=UTF-8`)
	if idx := strings.IndexRune(ct, ';'); idx >= 0 {
		ct = ct[:idx]
	}

	// check proto from request
	switch ct {
	case "application/json":
		op = ws.OpText
	default:
		op = ws.OpBinary
	}

	hdr := make(http.Header)
	if proto, ok := r.Header["Sec-WebSocket-Protocol"]; ok {
		for _, p := range proto {
			switch p {
			case "binary":
				hdr["Sec-WebSocket-Protocol"] = []string{"binary"}
				op = ws.OpBinary
			}
		}
	}
	payload, err := requestPayload(r)
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(err)
		}
		return
	}

	// We are using default parameters here since we use
	// wsflate.{Compress,Decompress}Frame helpers below in the code.
	// This assumes that we use standard compress/flate package as flate
	// implementation.
	e := wsflate.Extension{
		Parameters: wsflate.DefaultParameters,
	}
	upgrader := ws.HTTPUpgrader{
		Timeout: 5 * time.Second,
		Protocol: func(proto string) bool {
			if strings.Contains(proto, "binary") {
				return true
			}
			// fallback to support all protocols now
			return true
		},
		Negotiate: e.Negotiate,
		Header:    hdr,
	}

	conn, rw, _, err := upgrader.Upgrade(r, w)
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(err)
		}
		return
	}

	_, compression := e.Accepted()

	defer func() {
		if err := conn.Close(); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
	}()

	var request interface{}
	if !bytes.Equal(payload, []byte(`{}`)) {
		switch ct {
		case "application/json", "":
			m := json.RawMessage(payload)
			request = &m
		default:
			request = &raw.Frame{Data: payload}
		}
	}

	// we always need to set content type for message
	if ct == "" {
		ct = "application/json"
	}
	req := c.NewRequest(
		service.Name,
		service.Endpoint.Name,
		request,
		client.WithContentType(ct),
		client.StreamingRequest(),
	)

	so := selector.WithStrategy(strategy(service.Services))
	// create a new stream
	stream, err := c.Stream(ctx, req, client.WithSelectOption(so))
	if err != nil {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(err)
		}
		return
	}

	if request != nil {
		if err = stream.Send(request); err != nil {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Error(err)
			}
			return
		}
	}

	writeLock := new(sync.Mutex)
	go writeLoop(rw, stream, writeLock)

	rsp := stream.Response()

	// receive from stream and send to client
	for {
		select {
		case <-ctx.Done():
			return
		case <-stream.Context().Done():
			return
		default:
			// read backend response body
			buf, err := rsp.Read()
			if err != nil {
				// wants to avoid import  grpc/status.Status
				if strings.Contains(err.Error(), "context canceled") {
					return
				}
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Error(err)
				}
				return
			}

			var frame ws.Frame
			if op == ws.OpText {
				frame = ws.NewTextFrame(buf)
			} else {
				frame = ws.NewBinaryFrame(buf)
			}
			if compression {
				frame, err = wsflate.CompressFrame(frame)
				if err != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Error(err)
					}
					return
				}
			}

			// write the response
			writeLock.Lock()
			err = ws.WriteFrame(rw, frame)
			writeLock.Unlock()
			if err != nil {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Error(err)
				}
				return
			}
			if err = rw.Flush(); err != nil {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Error(err)
				}
				return
			}
		}
	}
}

// writeLoop
func writeLoop(rw io.ReadWriter, stream client.Stream, writeLock *sync.Mutex) {
	// close stream when done
	defer stream.Close()

	for {
		select {
		case <-stream.Context().Done():
			return
		default:
			frame, err := ws.ReadFrame(rw)
			if err != nil {
				if wserr, ok := err.(wsutil.ClosedError); ok {
					switch wserr.Code {
					case ws.StatusGoingAway:
						// this happens when user leave the page
						return
					case ws.StatusNormalClosure, ws.StatusNoStatusRcvd:
						// this happens when user close ws connection, or we don't get any status
						return
					}
				}
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Error(err)
				}
				return
			}

			frame = ws.UnmaskFrameInPlace(frame)
			compressedMessage, err := wsflate.IsCompressed(frame.Header)
			if err != nil {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Error(err)
				}
				return
			}
			// Note that even after successful negotiation of
			// compression extension, both sides are able to send
			// non-compressed messages.
			if compressedMessage {
				frame, err = wsflate.DecompressFrame(frame)
				if err != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Error(err)
					}
					return
				}
			}

			switch frame.Header.OpCode {
			default:
				// not relevant
				continue
			case ws.OpPing:
				f := ws.NewPongFrame(frame.Payload)
				if compressedMessage {
					if f, err = wsflate.CompressFrame(f); err != nil {
						if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
							logger.Error(fmt.Errorf("ws compress poing failed: %w", err))
						}
						return
					}
				}
				writeLock.Lock()
				err = ws.WriteFrame(rw, f)
				writeLock.Unlock()
				if err != nil {
					if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
						logger.Error(fmt.Errorf("ws write poing failed: %w", err))
					}
					return
				}
				continue
			case ws.OpText, ws.OpBinary:
				break
			}
			// send to backend
			// default to trying json
			// if the extracted payload isn't empty lets use it
			request := &raw.Frame{Data: frame.Payload}
			if err = stream.Send(request); err != nil {
				if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
					logger.Error(err)
				}
				return
			}
		}
	}
}

func isStream(r *http.Request, srv *api.Service) bool {
	// check if it's a web socket
	if !isWebSocket(r) {
		return false
	}
	// check if the endpoint supports streaming
	for _, service := range srv.Services {
		for _, ep := range service.Endpoints {
			// skip if it doesn't match the name
			if ep.Name != srv.Endpoint.Name {
				continue
			}
			// matched if the name
			if v := ep.Metadata["stream"]; v == "true" {
				return true
			}
		}
	}
	return false
}

func isWebSocket(r *http.Request) bool {
	contains := func(key, val string) bool {
		vv := strings.Split(r.Header.Get(key), ",")
		for _, v := range vv {
			if val == strings.ToLower(strings.TrimSpace(v)) {
				return true
			}
		}
		return false
	}

	if contains("Connection", "upgrade") && contains("Upgrade", "websocket") {
		return true
	}

	return false
}
