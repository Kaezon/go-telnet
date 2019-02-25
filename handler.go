package telnet

// A Handler serves a TELNET (or TELNETS) connection.
//
// A handler implementation uses the passed client context to process
// incomming data and return appropriate feedback.
type Handler interface {
	ServeTELNET(Context)
}
