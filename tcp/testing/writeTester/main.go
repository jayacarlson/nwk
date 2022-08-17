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

// Super Simple Server for testing writing, waits for connection then sends a string back
//		Works with readTester or clientTester

var (
	lstatPipe = make(chan string) // local status pipe
	cstatPipe = make(chan string) // connection status pipe (hey, I sent something)
	errPipe   = make(chan error)  // global error pipe
	sysXit    = make(chan os.Signal, 1)
	ipport    string
	byWriter  bool
)

func init() {
	flag.BoolVar(&byWriter, "w", false, "Use Writer and not ConnWriter")
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

const recordStr = `----<start>This is data set #1 inside a pseudo record<end>----<start>This is the second data set inside a pseudo record<end>----<start>This is some data inside a pseudo record.`
const tailStr = `Have a nice day!`

func writeToConn(conn net.Conn) error {
	rw := tcp.NewReadWriter(conn)
	defer rw.Close() // or conn.Close()

	msg := fmt.Sprintf("Hello there, how are you today?\nXYZABCXYZABCXYZ%s*%s\n", recordStr, tailStr)
	err := rw.WriteString(msg)
	if nil != err {
		errPipe <- err
	}
	cstatPipe <- fmt.Sprintf("sent message to client")
	return err
}

func connHandler(cn int, serving string, rw tcp.ReadWriter) error {
	msg := fmt.Sprintf("Hello %s, you are connection #%d\nXYZABCXYZABCXYZ%s*%s\n", serving, cn, recordStr, tailStr)
	err := rw.WriteString(msg)
	if nil != err {
		errPipe <- err
	}
	cstatPipe <- fmt.Sprintf("sent msg to con#%d@%s", cn, serving)
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
		if byWriter {
			conn, _, err := l.WaitOnConnection()
			if nil != err {
				dbg.Error("WaitOnConnection: %v", err)
				break
			}
			err = writeToConn(conn)
			if nil != err {
				dbg.Error("writeToConn: %v", err)
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
