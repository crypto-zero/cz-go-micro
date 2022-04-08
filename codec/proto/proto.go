// Package proto provides a proto codec
package proto

import (
	"io"
	"io/ioutil"

	"c-z.dev/go-micro/codec"

	"google.golang.org/grpc/encoding"
	gep "google.golang.org/grpc/encoding/proto"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	if b == nil {
		return nil
	}
	buf, err := ioutil.ReadAll(c.Conn)
	if err != nil {
		return err
	}
	cc := encoding.GetCodec(gep.Name)
	return cc.Unmarshal(buf, b)
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	cc := encoding.GetCodec(gep.Name)
	buf, err := cc.Marshal(b)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(buf)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "proto"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}
