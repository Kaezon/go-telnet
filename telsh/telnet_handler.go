package telsh

import (
	"go-telnet"

	"github.com/reiver/go-oi"

	"bytes"
	"io"
	"strings"
	"sync"
)

const (
	defaultExitCommandName = "exit"
	defaultPrompt          = "§ "
	defaultWelcomeMessage  = "\r\nWelcome!\r\n"
	defaultExitMessage     = "\r\nGoodbye!\r\n"
)

type ShellHandler struct {
	muxtex       sync.RWMutex
	producers    map[string]Producer
	elseProducer Producer

	ExitCommandName string
	Prompt          string
	WelcomeMessage  string
	ExitMessage     string
}

func NewShellHandler() *ShellHandler {
	producers := map[string]Producer{}

	telnetHandler := ShellHandler{
		producers: producers,

		Prompt:          defaultPrompt,
		ExitCommandName: defaultExitCommandName,
		WelcomeMessage:  defaultWelcomeMessage,
		ExitMessage:     defaultExitMessage,
	}

	return &telnetHandler
}

func (telnetHandler *ShellHandler) Register(name string, producer Producer) error {

	telnetHandler.muxtex.Lock()
	telnetHandler.producers[name] = producer
	telnetHandler.muxtex.Unlock()

	return nil
}

func (telnetHandler *ShellHandler) MustRegister(name string, producer Producer) *ShellHandler {
	if err := telnetHandler.Register(name, producer); nil != err {
		panic(err)
	}

	return telnetHandler
}

func (telnetHandler *ShellHandler) RegisterHandlerFunc(name string, handlerFunc HandlerFunc) error {

	produce := func(ctx telnet.Context, name string, args ...string) Handler {
		return PromoteHandlerFunc(handlerFunc, args...)
	}

	producer := ProducerFunc(produce)

	return telnetHandler.Register(name, producer)
}

func (telnetHandler *ShellHandler) MustRegisterHandlerFunc(name string, handlerFunc HandlerFunc) *ShellHandler {
	if err := telnetHandler.RegisterHandlerFunc(name, handlerFunc); nil != err {
		panic(err)
	}

	return telnetHandler
}

func (telnetHandler *ShellHandler) RegisterElse(producer Producer) error {

	telnetHandler.muxtex.Lock()
	telnetHandler.elseProducer = producer
	telnetHandler.muxtex.Unlock()

	return nil
}

func (telnetHandler *ShellHandler) MustRegisterElse(producer Producer) *ShellHandler {
	if err := telnetHandler.RegisterElse(producer); nil != err {
		panic(err)
	}

	return telnetHandler
}

func (telnetHandler *ShellHandler) ServeTELNET(ctx telnet.Context) {

	logger := ctx.Logger
	if nil == logger {
		logger = internalDiscardLogger{}
	}

	colonSpaceCommandNotFoundEL := []byte(": command not found\r\n")

	var prompt bytes.Buffer
	var exitCommandName string
	var welcomeMessage string
	var exitMessage string

	prompt.WriteString(telnetHandler.Prompt)

	promptBytes := prompt.Bytes()

	exitCommandName = telnetHandler.ExitCommandName
	welcomeMessage = telnetHandler.WelcomeMessage
	exitMessage = telnetHandler.ExitMessage

	if err := ctx.LongWriteString(welcomeMessage); nil != err {
		logger.Errorf("Problem long writing welcome message: %v", err)
		return
	}
	logger.Debugf("Wrote welcome message: %q.", welcomeMessage)
	if err := ctx.LongWrite(promptBytes); nil != err {
		logger.Errorf("Problem long writing prompt: %v", err)
		return
	}
	logger.Debugf("Wrote prompt: %q.", promptBytes)

	var buffer [1]byte // Seems like the length of the buffer needs to be small, otherwise will have to wait for buffer to fill up.
	p := buffer[:]

	var line bytes.Buffer

	for {
		// Read 1 byte.
		n, err := ctx.Read(p)
		if n <= 0 && nil == err {
			continue
		} else if n <= 0 && nil != err {
			break
		}

		line.WriteByte(p[0])
		//logger.Tracef("Received: %q (%d).", p[0], p[0])

		if '\n' == p[0] {
			lineString := line.String()

			if "\r\n" == lineString {
				line.Reset()
				if err := ctx.LongWrite(promptBytes); nil != err {
					return
				}
				continue
			}

			//@TODO: support piping.
			fields := strings.Fields(lineString)
			logger.Debugf("Have %d tokens.", len(fields))
			logger.Tracef("Tokens: %v", fields)
			if len(fields) <= 0 {
				line.Reset()
				if err := ctx.LongWrite(promptBytes); nil != err {
					return
				}
				continue
			}

			field0 := fields[0]

			if exitCommandName == field0 {
				ctx.LongWriteString(exitMessage)
				return
			}

			var producer Producer

			telnetHandler.muxtex.RLock()
			var ok bool
			producer, ok = telnetHandler.producers[field0]
			telnetHandler.muxtex.RUnlock()

			if !ok {
				telnetHandler.muxtex.RLock()
				producer = telnetHandler.elseProducer
				telnetHandler.muxtex.RUnlock()
			}

			if nil == producer {
				//@TODO: Don't convert that to []byte! think this creates "garbage" (for collector).
				ctx.LongWrite([]byte(field0))
				ctx.LongWrite(colonSpaceCommandNotFoundEL)
				line.Reset()
				if err := ctx.LongWrite(promptBytes); nil != err {
					return
				}
				continue
			}

			handler := producer.Produce(ctx, field0, fields[1:]...)
			if nil == handler {
				ctx.LongWrite([]byte(field0))
				//@TODO: Need to use a different error message.
				ctx.LongWrite(colonSpaceCommandNotFoundEL)
				line.Reset()
				ctx.LongWrite(promptBytes)
				continue
			}

			//@TODO: Wire up the stdin, stdout, stderr of the handler.

			if stdoutPipe, err := handler.StdoutPipe(); nil != err {
				//@TODO:
			} else if nil == stdoutPipe {
				//@TODO:
			} else {
				connect(ctx, ctx.Writer, stdoutPipe)
			}

			if stderrPipe, err := handler.StderrPipe(); nil != err {
				//@TODO:
			} else if nil == stderrPipe {
				//@TODO:
			} else {
				connect(ctx, ctx.Writer, stderrPipe)
			}

			if err := handler.Run(); nil != err {
				//@TODO:
			}
			line.Reset()
			if err := ctx.LongWrite(promptBytes); nil != err {
				return
			}
		}

		//@TODO: Are there any special errors we should be dealing with separately?
		if nil != err {
			break
		}
	}

	ctx.LongWriteString(exitMessage)
	return
}

func connect(ctx telnet.Context, writer io.Writer, reader io.Reader) {

	logger := ctx.Logger

	go func(logger telnet.Logger) {

		var buffer [1]byte // Seems like the length of the buffer needs to be small, otherwise will have to wait for buffer to fill up.
		p := buffer[:]

		for {
			// Read 1 byte.
			n, err := reader.Read(p)
			if n <= 0 && nil == err {
				continue
			} else if n <= 0 && nil != err {
				break
			}

			//logger.Tracef("Sending: %q.", p)
			//@TODO: Should we be checking for errors?
			oi.LongWrite(writer, p)
			//logger.Tracef("Sent: %q.", p)
		}
	}(logger)
}
