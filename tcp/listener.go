package tcp

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk"
)

var ListenDbg = dbg.Dbg{false, 0}

/*
	Routines to create server listeners, or able to use the ReadWriter code
		indirectly from HandleRequest(s) or implement your own more complex
		code using WaitOnConnection to use the net.Conn directly

		NewListener( ListenIP, Status chan ) ( *Listener, error ):
			TCP utility to wait on TCP connections
				ListenIP: local interface ip:port to listen on
				Status:   Channel to receive status messages:
							"Listener Started" on startup
							"Listener Waiting" each WaitOnConnection
							"Listener Closed" finally sent on close
						  If using the HandleRequest(s) functions,
						  any Con and Dis messages are also output:
						    Con<connection#>@<clientIP>
					  			e.g. Con15@127.0.0.1:47556
						    Dis<connection#>@<clientIP>(<resultErr>)
								e.g. Dis15@127.0.0.1:47556(EOF)

		Listener.Close():
			Close the listener

		Listener.Counts() ( serving, totalConnections int ):
			Returns the current number of connections being served
			and the total number of connections handled

		Listener.WaitOnConnection() ( net.Conn, int, error ):
			Waits until request made on listener returns
				net.Conn which MUST BE CLOSED BY THE USER
				the connection number (for ref only)
				any error
			Retuns a net.Conn and not a ReadWriter in case you
			want to do something more complex with the net.Conn.
			For simplicity there is the ConnHandler type which
			will handle simple read/writes by returning a
			simple ReadWriter and will close the conn afterwords.

		Listener.HandleRequests( ConnHandler, ErrPipe ) error:
			Routine to handle repeated connections
			Can be used as a GO ROUTINE or not
			Aborts and returns any error received from
			making a WaitOnConnection call -- e.g. closing
			the Listener
			Spawns a new GO ROUTINE for each connection
			with the ConnHandler func
			Any error received by the ConnHandler will be
				sent through any supplied ErrPipe

		Listener.HandleARequest( ConnHandler ) error:
			Routine to handle a single connection
			Returns any error received from WaitOnConnect
				call; otherwise the result from the
				ConnHandler after routing through the
				local handleConn
*/

type (
	Listener struct {
		connections uint32           // total number of connections made
		servicing   uint32           // number of active connections, updated by handleConn func
		hostIP      string           // host IP and port
		statPipe    chan<- string    // chan for any status output
		timeout     time.Duration    // listen timeout for WaitOnConnect
		listener    *net.TCPListener // actual TCP listener
	}
)

// create a TCP listener, waiting on connections from remote (client) PCs
func NewListener(ipPort string, status chan<- string) (*Listener, error) {
	l := Listener{hostIP: ipPort, statPipe: status}

	tcpa, err := net.ResolveTCPAddr("tcp", ipPort)
	if ListenDbg.ChkErr(err) {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", tcpa)
	if ListenDbg.ChkErr(err) {
		return nil, err
	}
	l.listener = listener
	l.status("Listener Created")

	return &l, nil
}

// ========================================================================= //

// Set the listen timeout for new connections
func (l *Listener) SetTimeout(timeout time.Duration) {
	l.timeout = timeout
}

// Close the listener
func (l *Listener) Close() {
	l.listener.Close()
	l.status("Listener Closed")
}

// Return the current and total number of connections
func (l *Listener) Counts() (servicing, totalConnections int) {
	return int(atomic.LoadUint32(&l.servicing)), int(atomic.LoadUint32(&l.connections))
}

/*
	Waits on a connection through the Listener, returns:
		on success the net.Conn, and the connection count
		on failure, returns the error why

	Caller is responsible for calling the Close on the net.Conn
*/
func (l *Listener) WaitOnConnection() (net.Conn, int, error) {
	expiry := time.Time{}
	if l.timeout != 0 {
		expiry = time.Now().Add(l.timeout)
	}
	l.listener.SetDeadline(expiry)
	l.status("Listener Waiting")
	conn, err := l.listener.Accept()
	err = nwk.ChkNetErr(err)
	if ListenDbg.ChkErrI(err, []error{nwk.Err_NoConnection}) {
		return nil, -1, err
	}
	atomic.AddUint32(&l.servicing, 1)
	return conn, int(atomic.AddUint32(&l.connections, 1)), nil
}

/*
	Handle repeated connection requests (mainly as a GO ROUTINE)

	Calls WaitOnConnection, and on a connection will route the
		connection to the ConnHandler via handleConn, which
		will then close the net.Conn

	Sends any error from the ConnHandler through the errPipe
	Exits and returns any error received from the WaitOnConn
	e.g. "Timedout" or "Connection not open"
*/
func (l *Listener) HandleRequests(ch ConnHandler, errPipe chan<- error) error {
	for {
		conn, conNum, err := l.WaitOnConnection()
		if err != nil {
			return err
		}
		go l.handleConn(conn, conNum, ch, errPipe)
	}
}

/*
	Handle a single connection request

	Calls WaitOnConnection, and on a connection will route the
		connection to the ConnHandler via handleConn, which
		will then close the net.Conn, and return any error
		from the ConnHandler
	Exits and returns any error received from the WaitOnConn
	e.g. "Timedout" or "Connection not open"
*/
func (l *Listener) HandleARequest(ch ConnHandler) error {
	conn, conNum, err := l.WaitOnConnection()
	if err != nil {
		return err // Timedout or Connection not open
	}
	return l.handleConn(conn, conNum, ch, nil)
}

// ------------------------------------------------------------------------- //

func (l *Listener) handleConn(conn net.Conn, conNum int, ch ConnHandler, errPipe chan<- error) error {
	serving := conn.RemoteAddr().String()
	rw := NewReadWriter(conn)
	l.status(fmt.Sprintf("Con%d@%s", conNum, serving))
	err := nwk.ChkNetErr(ch(conNum, serving, rw))
	conn.Close() // the rw is closed in-effect when the conn is closed
	if nil != err && nil != errPipe {
		errPipe <- err
	}
	atomic.AddUint32(&l.servicing, ^uint32(0)) // -1 w/o error
	l.status(fmt.Sprintf("Dis%d@%s(%v)", conNum, serving, err))
	return err
}

func (l *Listener) status(s string) {
	if nil != l.statPipe {
		l.statPipe <- s
	}
}
