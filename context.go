package telnet

import (
	"net"

	"github.com/reiver/go-oi"
)

type Context struct {
	Logger     Logger
	Connection net.Conn
	Reader     *internalDataReader
	Writer     *internalDataWriter
}

func NewContext(conn net.Conn, reader *internalDataReader, writer *internalDataWriter) *Context {
	ctx := Context{Connection: conn, Reader: reader, Writer: writer}

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

func (ctx *Context) LongWrite(data []byte) {
	oi.LongWrite(ctx.Writer, data)
}
