package grpc

import (
	"encoding/json"
	"errors"
	"strings"

	"c-z.dev/go-micro/codec"
	"c-z.dev/go-micro/codec/bytes"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	gep "google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ErrInvalidMessage = errors.New("invalid message")

type (
	jsonCodec  struct{}
	bytesCodec struct{}
	wrapCodec  struct{ encoding.Codec }
)

var jsonpbMarshaler = &protojson.MarshalOptions{
	UseProtoNames:  true,
	UseEnumNumbers: false,
	AllowPartial:   false,
}

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
	if v == nil {
		return nil
	}
	return w.Codec.Unmarshal(data, v)
}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		return jsonpbMarshaler.Marshal(pb)
	}

	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if pb, ok := v.(proto.Message); ok {
		return protojson.Unmarshal(data, pb)
	}
	return json.Unmarshal(data, v)
}

func (jsonCodec) Name() string {
	return "json"
}

func (bytesCodec) Marshal(v interface{}) ([]byte, error) {
	b, ok := v.(*[]byte)
	if !ok {
		return nil, ErrInvalidMessage
	}
	return *b, nil
}

func (bytesCodec) Unmarshal(data []byte, v interface{}) error {
	b, ok := v.(*[]byte)
	if !ok {
		return ErrInvalidMessage
	}
	*b = data
	return nil
}

func (bytesCodec) Name() string {
	return "bytes"
}

type grpcCodec struct {
	// headers
	id       string
	target   string
	method   string
	endpoint string

	s grpc.ServerStream
	c encoding.Codec
}

func (g *grpcCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	md, _ := metadata.FromIncomingContext(g.s.Context())
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
	// caller has requested a frame
	if f, ok := v.(*bytes.Frame); ok {
		return g.s.RecvMsg(f)
	}
	return g.s.RecvMsg(v)
}

func (g *grpcCodec) Write(m *codec.Message, v interface{}) error {
	// if we don't have a body
	if v != nil {
		b, err := g.c.Marshal(v)
		if err != nil {
			return err
		}
		m.Body = b
	}
	// write the body using the framing codec
	return g.s.SendMsg(&bytes.Frame{Data: m.Body})
}

func (g *grpcCodec) Close() error {
	return nil
}

func (g *grpcCodec) String() string {
	return "grpc"
}
