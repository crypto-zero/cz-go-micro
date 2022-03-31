package grpc

import (
	"strings"

	"c-z.dev/go-micro/codec"
	"c-z.dev/go-micro/codec/bytes"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

type response struct {
	conn   *grpc.ClientConn
	stream grpc.ClientStream
	codec  encoding.Codec
	gcodec codec.Codec
}

// Codec read the response
func (r *response) Codec() codec.Reader {
	return r.gcodec
}

// Header read the header
func (r *response) Header() map[string]string {
	md, err := r.stream.Header()
	if err != nil {
		return map[string]string{}
	}
	hdr := make(map[string]string, len(md))
	for k, v := range md {
		hdr[k] = strings.Join(v, ",")
	}
	return hdr
}

// Read the undecided response
func (r *response) Read() ([]byte, error) {
	f := &bytes.Frame{}
	if err := r.gcodec.ReadBody(f); err != nil {
		return nil, err
	}
	return f.Data, nil
}
