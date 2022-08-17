package nwk

import (
	"errors"
	"io"
	"net"
	"os"
	"syscall"

	"github.com/jayacarlson/dbg"
)

var (
	// Network errors
	Err_UnknownHost       = errors.New("Unknown host")
	Err_AddrNotFound      = errors.New("Addr not found")
	Err_ConnectionRefused = errors.New("Connection refused")
	Err_NoConnection      = errors.New("Connection not open")
	Err_LostConnection    = errors.New("Lost connection") // same as EOF
	Err_ClosedByUser      = errors.New("Closed by user")
	Err_UserExit          = errors.New("User exit request")
	Err_Timeout           = errors.New("Timedout")
	Err_ResetByPeer       = errors.New("Reset by peer")
	Err_ClosedRemotely    = errors.New("Broken pipe, closed remotely")
	Err_EndOfFile         = errors.New("End of file/data")
	Err_NoData            = errors.New("No data received")
	Err_BadData           = errors.New("Bad data received")
	Err_AddressInUse      = errors.New("Address in use")
	Err_IllegalParam      = errors.New("Illegal/missing param")
	Err_BadInterface      = errors.New("Unknown interface")
)

func netErr(oerr, err error) error {
	switch t := err.(type) {
	case *net.OpError:
		return netErr(oerr, t.Err)
	case *os.SyscallError:
		return netErr(oerr, t.Err)
	case syscall.Errno:
		switch t {
		case syscall.ECONNREFUSED:
			return Err_ConnectionRefused
		case syscall.EHOSTUNREACH:
			return Err_UnknownHost
		case syscall.ECONNRESET:
			return Err_ResetByPeer
		case syscall.EPIPE:
			return Err_ClosedRemotely
		case syscall.EADDRINUSE:
			return Err_AddressInUse
		default:
			dbg.Message("nwk.netErr - syscall.Errno: %03x  '%v'", int(t), oerr)
		}
	default:
		if err.Error() == "use of closed network connection" { // better way to catch this?
			return Err_NoConnection
		}
		dbg.Message("nwk.netErr - unknown type: %v", t)
	}
	return oerr
}

func ChkNetErr(err error) error {
	if err != nil {
		if err == io.EOF {
			return err
		}
		if netError, ok := err.(net.Error); ok {
			if netError.Timeout() {
				return Err_Timeout
			}
			return netErr(err, err)
		}
	}
	return err
}
