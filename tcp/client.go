package tcp

import (
	"net"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk"
)

var ClientDbg = dbg.Dbg{false, 0}

/*
	Simple TCP client; connects to server and can read from / write to it,
		uses ReadWriter or ReadBufWriter depending on params

		NewClient( srvrPort, timeout, buffered ) ( *ReadWriter, error ):
			Create a new ReadWriter connected to requested server
				srvrPort:	serverIP:port attempting to connect with
				timeout:	a timeout if desired
				buffered:	return a buffered writer in the ReadWriter
			On connection returns ReadWriter, or returns error
	User must Close the client (ReadWriter)
*/

func NewClient(srvrPort string, timeout time.Duration, buf bool) (ReadWriter, error) {
	var x ReadWriter
	var conn net.Conn
	var err error

	if 0 == timeout {
		conn, err = net.Dial("tcp", srvrPort)
	} else {
		conn, err = net.DialTimeout("tcp", srvrPort, timeout)
	}
	if err = nwk.ChkNetErr(err); nil != err {
		ClientDbg.Error("netDial failed: %v", err)
		return nil, err
	}
	ClientDbg.Info("Connection made to: %s", srvrPort)

	if buf {
		// return buffered reads & writes
		rw := newReadBufWriter(conn)
		rw.r.srvrIP = srvrPort
		x = rw
	} else {
		// return just buffered reads
		rw := newReadWriter(conn)
		rw.srvrIP = srvrPort
		x = rw
	}
	return x, nil
}
