package telnet

import (
	"io"
	"net"

	"github.com/reiver/go-oi"
)

type Context struct {
	Logger     Logger
	Connection net.Conn
	Reader     *internalDataReader
	Writer     *internalDataWriter
}

func NewContext(conn net.Conn, reader io.Reader, writer io.Writer) *Context {
	ctx := Context{Connection: conn, Reader: newDataReader(reader), Writer: newDataWriter(writer)}

	return &ctx
}

func (ctx *Context) InjectLogger(logger Logger) *Context {
	ctx.Logger = logger

	return ctx
}

func (ctx *Context) Read(data []byte) (n int, err error) {
	return ctx.Reader.Read(data)
}

func (ctx *Context) Write(data []byte) (n int, err error) {
	return ctx.Writer.Write(data)
}

func (ctx *Context) LongWrite(data []byte) error {
	_, err := oi.LongWrite(ctx.Writer, data)
	return err
}

func (ctx *Context) LongWriteString(data string) error {
	_, err := oi.LongWriteString(ctx.Writer, data)
	return err
}
