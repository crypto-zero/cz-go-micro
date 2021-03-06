package grpc

import (
	b "bytes"
	"encoding/json"
	"fmt"
	"strings"

	"c-z.dev/go-micro/codec"
	"c-z.dev/go-micro/codec/bytes"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	gep "google.golang.org/grpc/encoding/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type (
	jsonCodec  struct{}
	bytesCodec struct{}
	wrapCodec  struct{ encoding.Codec }
)

var useNumber bool

var defaultGRPCCodecs = map[string]encoding.Codec{
	"application/json":         jsonCodec{},
	"application/proto":        encoding.GetCodec(gep.Name),
	"application/protobuf":     encoding.GetCodec(gep.Name),
	"application/octet-stream": encoding.GetCodec(gep.Name),
	"application/grpc":         encoding.GetCodec(gep.Name),
	"application/grpc+json":    jsonCodec{},
	"application/grpc+proto":   encoding.GetCodec(gep.Name),
	"application/grpc+bytes":   bytesCodec{},
}

// UseNumber fix unmarshal Number(8234567890123456789) to interface(8.234567890123457e+18)
func UseNumber() {
	useNumber = true
}

func (w wrapCodec) String() string {
	return w.Codec.Name()
}

func (w wrapCodec) Marshal(v interface{}) ([]byte, error) {
	b, ok := v.(*bytes.Frame)
	if ok {
		return b.Data, nil
	}
	return w.Codec.Marshal(v)
}

func (w wrapCodec) Unmarshal(data []byte, v interface{}) error {
	b, ok := v.(*bytes.Frame)
	if ok {
		b.Data = data
		return nil
	}
	return w.Codec.Unmarshal(data, v)
}

func (bytesCodec) Marshal(v interface{}) ([]byte, error) {
	b, ok := v.(*[]byte)
	if !ok {
		return nil, fmt.Errorf("failed to marshal: %v is not type of *[]byte", v)
	}
	return *b, nil
}

func (bytesCodec) Unmarshal(data []byte, v interface{}) error {
	b, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal: %v is not type of *[]byte", v)
	}
	*b = data
	return nil
}

func (bytesCodec) Name() string {
	return "bytes"
}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	if b, ok := v.(*bytes.Frame); ok {
		return b.Data, nil
	}

	if pb, ok := v.(proto.Message); ok {
		data, err := protojson.Marshal(pb)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if b, ok := v.(*bytes.Frame); ok {
		b.Data = data
		return nil
	}
	if pb, ok := v.(proto.Message); ok {
		return protojson.Unmarshal(data, pb)
	}

	dec := json.NewDecoder(b.NewReader(data))
	if useNumber {
		dec.UseNumber()
	}
	return dec.Decode(v)
}

func (jsonCodec) Name() string {
	return "json"
}

type grpcCodec struct {
	// headers
	id       string
	target   string
	method   string
	endpoint string

	s grpc.ClientStream
	c encoding.Codec
}

func (g *grpcCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	md, err := g.s.Header()
	if err != nil {
		return err
	}
	if m == nil {
		m = new(codec.Message)
	}
	if m.Header == nil {
		m.Header = make(map[string]string, len(md))
	}
	for k, v := range md {
		m.Header[k] = strings.Join(v, ",")
	}
	m.Id = g.id
	m.Target = g.target
	m.Method = g.method
	m.Endpoint = g.endpoint
	return nil
}

func (g *grpcCodec) ReadBody(v interface{}) error {
	if f, ok := v.(*bytes.Frame); ok {
		return g.s.RecvMsg(f)
	}
	return g.s.RecvMsg(v)
}

func (g *grpcCodec) Write(m *codec.Message, v interface{}) error {
	// if we don't have a body
	if v != nil {
		return g.s.SendMsg(v)
	}
	// write the body using the framing codec
	return g.s.SendMsg(&bytes.Frame{Data: m.Body})
}

func (g *grpcCodec) Close() error {
	return g.s.CloseSend()
}

func (g *grpcCodec) String() string {
	return g.c.Name()
}
