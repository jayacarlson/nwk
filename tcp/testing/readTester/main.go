package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk"
	"github.com/jayacarlson/nwk/tcp"
)

// Super Simple Server for testing reading, waits for connection then reads the string sent
//		Works with readTester or clientTester

var (
	lstatPipe = make(chan string) // local status pipe
	cstatPipe = make(chan string) // connection status pipe (hey, I sent something)
	errPipe   = make(chan error)  // global error pipe
	sysXit    = make(chan os.Signal, 1)
	ipport    string
	byReader  bool
)

func init() {
	flag.BoolVar(&byReader, "r", false, "Use Reader and not ConnReader")
	flag.StringVar(&ipport, "use", "127:7879", "Interface & port to use")
}

func pipeReader() {
	dbg.Echo("pipeReader started")
	for {
		select {
		case l := <-lstatPipe:
			dbg.Message("%s", l)
		case w := <-cstatPipe:
			dbg.Echo("%s", w)
		case e := <-errPipe:
			dbg.Error("%v", e)
		case <-sysXit:
			dbg.Echo("losing pipeReader and all pipes...")
			close(lstatPipe)
			close(cstatPipe)
			close(errPipe)
			close(sysXit)
			time.Sleep(time.Millisecond * 250)
			dbg.Fatal("")
		}
	}
}

func readFromConn(conn net.Conn) error {
	rw := tcp.NewReadWriter(conn)
	defer rw.Close() // or conn.Close()

	msg, err := rw.ReadString()
	if nil != err {
		errPipe <- err
	}
	msg = strings.TrimSpace(msg)
	cstatPipe <- fmt.Sprintf("received `%s` from client", msg)
	return err
}

func connHandler(cn int, serving string, rw tcp.ReadWriter) error {
	msg, err := rw.ReadString()
	if nil != err {
		errPipe <- err
	}
	msg = strings.TrimSpace(msg)
	cstatPipe <- fmt.Sprintf("received `%s` from con#%d@%s", msg, cn, serving)
	return err
}

func main() {
	flag.Parse()

	signal.Notify(sysXit, syscall.SIGINT, syscall.SIGTERM)

	ip, err := nwk.FindMyIP4Addr(ipport)
	dbg.ChkErrX(err, "%v", err)
	dbg.ChkTruX(-1 != strings.Index(ip, ":"), "Must have valid port")
	dbg.Message("My IPAddr: %s", ip)

	go pipeReader()

	l, err := tcp.NewListener(ip, lstatPipe)
	if dbg.ChkErr(err, "Failed to get listener: %v", err) {
		return
	}

	defer l.Close()

	for {
		if byReader {
			conn, _, err := l.WaitOnConnection()
			if nil != err {
				dbg.Error("WaitOnConnection: %v", err)
				break
			}
			err = readFromConn(conn)
			if nil != err {
				dbg.Error("readFromConn: %v", err)
				break
			}
		} else {
			err := l.HandleARequest(connHandler)
			if nil != err {
				dbg.Error("HandleARequest: %v", err)
				break
			}
		}
	}
}
