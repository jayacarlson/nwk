package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk"
	"github.com/jayacarlson/nwk/tcp"
)

/*
	A Simple server for testing reading from / writing to
		Uses serverTester
*/

var (
	statPipe = make(chan string, 2) // listener status pipe
	errPipe  = make(chan error, 2)  // global error pipe
	sysXit   = make(chan os.Signal, 1)
	lights   = [3]string{"Red\n", "Yellow\n", "Green\n"}
	ipport   string
)

func init() {
	flag.StringVar(&ipport, "use", "127:7879", "Interface & port to use")
}

func connHandler(cn int, serving string, rw tcp.ReadWriter) error {
	var req string
	err := rw.WriteString(fmt.Sprintf("Hello %s, you are connection #%d\n", serving, cn))
	for {
		if nil != err {
			if io.EOF != err {
				errPipe <- err
			}
			return err
		}

		req, err = rw.ReadString()
		if nil == err {
			req = strings.TrimSpace(req)
			switch req {
			case "light":
				err = rw.WriteString(lights[rand.Intn(3)])
			case "byte":
				err = rw.WriteByte(byte(rand.Intn(255) + 1)) // 1..255
			case "done":
				err = nwk.Err_ClosedByUser
			default:
				if "I am: " == req[:6] {
					dbg.Echo("IAm: %s", req[6:])
				} else {
					err = errors.New(fmt.Sprintf("Unknown request: `%s`", req))
				}
			}
		}
	}
}

// GO function, waits on requests, handles them 1 at a time
func listener(l *tcp.Listener) {
	for {
		err := l.HandleARequest(connHandler)
		if nil != err {
			if io.EOF != err {
				if nwk.Err_NoConnection == err {
					dbg.Message("Exiting listener loop")
					break
				}
				dbg.Error("HandleReadWriterRequest: %v", err)
			}
		}
	}
}

// Server for testing simple requests with read/writes

func main() {
	flag.Parse()

	signal.Notify(sysXit, syscall.SIGINT, syscall.SIGTERM)

	ip, err := nwk.FindMyIP4Addr(ipport)
	dbg.ChkErrX(err, "%v", err)
	dbg.ChkTruX(-1 != strings.Index(ip, ":"), "Must have valid port")
	dbg.Message("My IPAddr: %s", ip)

	l, err := tcp.NewListener(ip, statPipe)
	dbg.ChkErrX(err, "Failed to get listener: %v", err)

	go listener(l)

loop:
	for {
		select {
		case s := <-statPipe:
			dbg.Info("%s", s)
		case e := <-errPipe:
			dbg.Error("errPipe: %v", e)
		case <-sysXit:
			dbg.Echo("losing the listener...")
			l.Close()
			close(sysXit)

			// wait for all connections to close
			ticker := time.NewTicker(time.Second / 4)
		waitLoop:
			for {
				select {
				case s := <-statPipe:
					dbg.Warning("%s", s)
				case e := <-errPipe:
					dbg.Error("errPipe: %v", e)
				case <-ticker.C:
					if c, _ := l.Counts(); 0 == c {
						break waitLoop
					} else {
						dbg.Note("== Waiting on %d connections to close. ==", c)
					}
				}
			}
			ticker.Stop()
			close(statPipe)
			close(errPipe)
			break loop
		}
	}
}
