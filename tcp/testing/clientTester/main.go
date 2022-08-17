package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk/tcp"
)

/*
	Client for testing reading / writing -- server sending messages
		Works with readTester or writeTester
		Can either send or receive messages
*/

var (
	server string
	send   bool
	bug    = dbg.Dbg{}
)

func init() {
	flag.StringVar(&server, "server", "", "Server to try reading from")
	flag.BoolVar(&send, "s", false, "Send data, dont read data")
}

var smlBuf = make([]byte, 3)

func trySending() error {
	c, err := tcp.NewClient(server, time.Second, false)
	if bug.ChkErr(err, "NewClient: %v", err) {
		return err
	}
	defer func() { c.Close(); time.Sleep(time.Second) }()

	msg := fmt.Sprintf("Hello %s, how are you today?\n", server)

	// 1st part is a string
	err = c.WriteString(msg)
	dbg.ChkErr(err, "WriteString error: %v", err)
	return err
}

func tryReading() error {
	c, err := tcp.NewClient(server, time.Second, false)
	if bug.ChkErr(err, "NewClient: %v", err) {
		return err
	}
	defer func() { c.Close(); time.Sleep(time.Second) }()

	// 1st part is a string
	s, err := c.ReadString()
	if !dbg.ChkErrI(err, []error{io.EOF}, "ReadString error: %v", err) {
		fmt.Printf("ReadString: `%s`\n", strings.TrimSpace(s))
	}
	// 2nd group is a series of 3 bytes
	z := 0
	for l := 0; nil == err && l < 5; l++ {
		z, err = c.Read(smlBuf)
		if !dbg.ChkErrI(err, []error{io.EOF}, "Read error: %v", err) {
			fmt.Printf("Read: (%d) `%s`\n", z, string(smlBuf))
		}
	}
	// do ReadRecord looking for <end>
	r1, err := c.ReadRecord(nil, []byte("<end>"))
	if !dbg.ChkErrI(err, []error{io.EOF}, "ReadRecord error: %v", err) {
		fmt.Printf("ReadRecord<end>: `%s`\n", string(r1))
	}
	// do ReadRecord looking for <start><end>
	r2, err := c.ReadRecord([]byte("<start>"), []byte("<end>"))
	if !dbg.ChkErrI(err, []error{io.EOF}, "ReadRecord error: %v", err) {
		fmt.Printf("<start>ReadRecord<end>: `%s`\n", string(r2))
	}
	// do ReadSizedRecord looking for <start>
	r3, err := c.ReadSizedRecord([]byte("<start>"), 41)
	if !dbg.ChkErrI(err, []error{io.EOF}, "ReadSizedRecord error: %v", err) {
		fmt.Printf("<start>ReadSizedRecord: `%s`\n", string(r3))
	}

	// read a single byte
	b, err := c.ReadByte()
	if !dbg.ChkErrI(err, []error{io.EOF}, "ReadByte error: %v", err) {
		fmt.Printf("ReadByte gave: %c (should be a '*')\n", rune(b))
	}

	// read trailing string
	s, err = c.ReadString()
	if !dbg.ChkErrI(err, []error{io.EOF}, "ReadString error: %v", err) {
		fmt.Printf("ReadString: `%s`\n", strings.TrimSpace(s))
	}

	if nil != err {
		dbg.Error(" err: %v", err)
	}
	return err
}

func main() {
	flag.Parse()

	dbg.ChkTruX("" != server, "Must supply -server <ip> to read from")

	tcp.ClientDbg.Enabled = true
	bug.Enabled = false

	for test := 0; test < 10; test++ {
		var err error
		if send {
			err = trySending()
		} else {
			err = tryReading()
		}
		if nil != err && io.EOF != err {
			break
		}
	}
}
