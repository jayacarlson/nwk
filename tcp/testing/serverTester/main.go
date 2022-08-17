package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk"
	"github.com/jayacarlson/nwk/tcp"
)

/*
	Client for testing with the simple server
		Uses simpleServer

*/

var server string

func init() {
	flag.StringVar(&server, "server", "", "Server to test")
}

func test1() error {
	dbg.Info("\n===== %s =====", dbg.IAm())
	// just connect and see if we get the 'Hello' message

	c, err := tcp.NewClient(server, time.Second, false)
	if dbg.ChkErr(err, "Failed creating server %v", err) {
		return err
	}
	defer func() { c.Close(); time.Sleep(time.Second) }()
	// read "Hello message"
	s, err := c.ReadString()
	if !dbg.ChkErr(err, "ReadString error: %v", err) {
		dbg.Echo("Hello String: `%s`", strings.TrimSpace(s))
		err = c.WriteString(fmt.Sprintf("I am: %s - simple read greeting\n", dbg.IAm()))
	}
	return err
}

func test2() error {
	dbg.Info("\n===== %s =====", dbg.IAm())
	// just connect and see if we get the 'Hello' message, but delay before closing

	c, err := tcp.NewClient(server, time.Second, false)
	if dbg.ChkErr(err, "Failed creating server %v", err) {
		return err
	}
	defer func() { time.Sleep(time.Second); c.Close() }()
	// read "Hello message"
	s, err := c.ReadString()
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}
	dbg.Echo("Hello String: `%s`", strings.TrimSpace(s))

	err = c.WriteString(fmt.Sprintf("I am: %s - read greeting and pause before close\n", dbg.IAm()))

	return err
}

func test3() error {
	dbg.Info("\n===== %s =====", dbg.IAm())
	// make request and close

	c, err := tcp.NewClient(server, time.Second, false)
	if dbg.ChkErr(err, "Failed creating server %v", err) {
		return err
	}
	defer func() { c.Close(); time.Sleep(time.Second) }()
	// read "Hello message"
	s, err := c.ReadString()
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}
	dbg.Echo("Hello String: `%s`", strings.TrimSpace(s))

	err = c.WriteString(fmt.Sprintf("I am: %s - request a light setting\n", dbg.IAm()))
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}

	// do string request of 'light'
	err = c.WriteString("light\n")
	if dbg.ChkErr(err, "WriteString error: %v", err) {
		return err
	}

	r, err := c.ReadString()
	if !dbg.ChkErr(err, "ReadString error: %v", err) {
		dbg.Echo("Light String: `%s`", strings.TrimSpace(r))
	}

	return err
}

func test4() error {
	dbg.Info("\n===== %s =====", dbg.IAm())
	// make request, but delay before close

	c, err := tcp.NewClient(server, time.Second, false)
	if dbg.ChkErr(err, "Failed creating server %v", err) {
		return err
	}
	defer func() { time.Sleep(time.Second); c.Close() }()
	// read "Hello message"
	s, err := c.ReadString()
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}
	dbg.Echo("Hello String: `%s`", strings.TrimSpace(s))

	err = c.WriteString(fmt.Sprintf("I am: %s - request a light setting, pause before close\n", dbg.IAm()))
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}

	// do string request of 'light'
	err = c.WriteString("light\n")
	if dbg.ChkErr(err, "WriteString error: %v", err) {
		return err
	}

	r, err := c.ReadString()
	if !dbg.ChkErr(err, "ReadString error: %v", err) {
		dbg.Echo("Light String: `%s`", strings.TrimSpace(r))
	}

	return err
}

func test5() error {
	dbg.Info("\n===== %s =====", dbg.IAm())
	// make request, delay, send 'done' then try reading

	c, err := tcp.NewClient(server, time.Second, false)
	if dbg.ChkErr(err, "Failed creating server %v", err) {
		return err
	}
	defer c.Close()
	// read "Hello message"
	s, err := c.ReadString()
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}
	dbg.Echo("Hello String: `%s`", strings.TrimSpace(s))

	err = c.WriteString(fmt.Sprintf("I am: %s - request a light setting, do user close, try reading\n", dbg.IAm()))
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}

	// do string request of 'light'
	err = c.WriteString("light\n")
	if dbg.ChkErr(err, "WriteString error: %v", err) {
		return err
	}

	r, err := c.ReadString()
	if !dbg.ChkErr(err, "ReadString error: %v", err) {
		dbg.Echo("Light String: `%s`", strings.TrimSpace(r))
	}

	err = c.WriteString("done\n")
	if dbg.ChkErr(err, "WriteString error: %v", err) {
		return err
	}

	time.Sleep(time.Second / 4)

	r, err = c.ReadString()
	if io.EOF != err {
		dbg.Error("ReadString error: `%v` -- EXPECTING `EOF`", err)
		dbg.Echo("SHOULD NOT HAVE RECEIVED THIS: `%s`", strings.TrimSpace(r))
	} else {
		err = nil
	}

	return err
}

func test6() error {
	dbg.Info("\n===== %s =====", dbg.IAm())
	// make request, delay, send 'done' then try writing again

	c, err := tcp.NewClient(server, time.Second, false)
	if dbg.ChkErr(err, "Failed creating server %v", err) {
		return err
	}
	defer c.Close()
	// read "Hello message"
	s, err := c.ReadString()
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}
	dbg.Echo("Hello String: `%s`", strings.TrimSpace(s))

	err = c.WriteString(fmt.Sprintf("I am: %s - request a light setting, do user close, and try writing again\n", dbg.IAm()))
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}

	// do string request of 'light'
	err = c.WriteString("light\n")
	if dbg.ChkErr(err, "WriteString error: %v", err) {
		return err
	}

	r, err := c.ReadString()
	if !dbg.ChkErr(err, "ReadString error: %v", err) {
		dbg.Echo("Light String: `%s`", strings.TrimSpace(r))
	}

	var i int
	for i = 0; i < 10; i++ {
		time.Sleep(time.Second / 2)
		err = c.WriteString("done\n")
		if nil != err {
			break
		}
	}
	if nwk.Err_ClosedRemotely != err {
		dbg.Error("WriteString error: `%v` -- EXPECTING `Broken pipe, closed remotely.`", err)
	} else {
		dbg.Echo("Conn closed at try %d", i)
		err = nil
	}
	return err
}

func test7() error {
	dbg.Info("\n===== %s =====", dbg.IAm())
	// make request, delay, send 'done' then try writing again

	c, err := tcp.NewClient(server, time.Second, false)
	if dbg.ChkErr(err, "Failed creating server %v", err) {
		return err
	}
	defer c.Close()
	// read "Hello message"
	s, err := c.ReadString()
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}
	dbg.Echo("Hello String: `%s`", strings.TrimSpace(s))

	err = c.WriteString(fmt.Sprintf("I am: %s - request a single byte\n", dbg.IAm()))
	if dbg.ChkErr(err, "ReadString error: %v", err) {
		return err
	}

	// do request of 'byte'
	err = c.WriteString("byte\n")
	if dbg.ChkErr(err, "WriteString error: %v", err) {
		return err
	}

	r, err := c.ReadByte()
	if !dbg.ChkErr(err, "ReadByte error: %v", err) {
		dbg.Echo("Byte Returned: `%x`", r)
	}

	return err
}

func main() {
	var err error

	flag.Parse()

	dbg.ChkTruX("" != server, "Must supply -sender <ip> to read from")

	tcp.ClientDbg.Enabled = true

	err = test1()
	dbg.ChkErr(err, "Test1 err: %v", err)
	err = test2()
	dbg.ChkErr(err, "Test2 err: %v", err)
	err = test3()
	dbg.ChkErr(err, "Test3 err: %v", err)
	err = test4()
	dbg.ChkErr(err, "Test4 err: %v", err)
	err = test5()
	dbg.ChkErr(err, "Test5 err: %v", err)
	err = test6()
	dbg.ChkErr(err, "Test6 err: %v", err)
	err = test7()
	dbg.ChkErr(err, "Test7 err: %v", err)

	dbg.Message("-- Exiting --")
}
