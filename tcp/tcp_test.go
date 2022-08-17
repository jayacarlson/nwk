package tcp

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk"
	"github.com/jayacarlson/tst"
)

var (
	chk       = tst.Chk{}
	sysXit    = make(chan os.Signal, 1) // system exit signal in case user wants to abort testing
	xitSig    = make(chan bool, 1)      // signal to pipeReader to exit
	tstatPipe = make(chan string, 8)    // test status pipe (buffered)
	sstatPipe = make(chan string)       // server status pipe
	serrPipe  = make(chan error)        // server error pipe
	cstatPipe = make(chan string)       // client status pipe
	cerrPipe  = make(chan error)        // client error pipe
	ticker    = time.NewTicker(time.Hour)

	loopback   = "127.0.0.1:1234"
	piperQuiet = false

	enableAll = true

	simpleListenerTests = (enableAll || true)
	simpleClientTests   = (enableAll || false)
	simpleReadTests     = (enableAll || false)
	clientReadTimeout   = (enableAll || false)
	multipleReadConns   = (enableAll || false)
	simpleWriteTests    = (enableAll || false)
	clientWriteTimeout  = (enableAll || false)
	multipleWriteConns  = (enableAll || false)
	testRecords         = (enableAll || false)
)

func pipeReader() {
	dbg.Caution("pipeReader started")
	for {
		select {
		case a := <-sstatPipe:
			if !piperQuiet {
				dbg.Message("ss: %s", a)
			}
		case b := <-serrPipe:
			if !piperQuiet {
				dbg.Error("se: %v", b)
			}
		case c := <-cstatPipe:
			if !piperQuiet {
				dbg.Echo("cs: %s", c)
			}
		case d := <-cerrPipe:
			if !piperQuiet {
				dbg.Warning("ce: %v", d)
			}
		case <-sysXit:
			dbg.Echo("losing pipeReader and all pipes...")
			closeAllPipes()
			time.Sleep(time.Millisecond * 250)
			dbg.Fatal("")
		case <-xitSig:
			dbg.Caution("pipeReader exiting")
			return
		}
	}
}

func closeAllPipes() {
	close(sstatPipe)
	close(cstatPipe)
	close(serrPipe)
	close(cerrPipe)
	close(xitSig)
	close(sysXit)
}

func waitFor(this string) error {
	ticker.Reset(time.Second * 5)
	for {
		select {
		case r := <-tstatPipe:
			if this == r {
				ticker.Reset(time.Hour)
				return nil
			}
			dbg.Caution("Waiting: %s", r)
		case <-ticker.C:
			dbg.Error("Expired, waiting for `%s`", this)
			ticker.Reset(time.Hour)
			return nwk.Err_Timeout
		}
	}
}

func init() {
	signal.Notify(sysXit, syscall.SIGINT, syscall.SIGTERM)
}

const story = `There once was a little old man who lived in a little old house.  
He had a little old dog, a little old cat and a little old mouse.  
In the evenings he and his pets would have their favorite meal.  
On Mondays he would take the little old dog on a nice long walk.  
On Tuesdays he would let the little old cat out to ramble about.  
On Wednesdays he would set the little old mouse out in the yard.  
On Thursdays and Fridays they would all watch the old television.  
Every weekend he would take them all over to the neigborhood park.  
He would sit on his bench and watch them all chase each other about.`

// ------------------------------------------------------------------------- //

func Test___init(_ *testing.T) {
	go pipeReader()
	time.Sleep(time.Millisecond * 100)
}

// ------------------------------------------------------------------------- //

func Test_SimpleOpenClose(t *testing.T) {
	tst.Testing("Simple Listener tests", "", simpleListenerTests)

	if simpleListenerTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"))
		time.Sleep(time.Second)
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Simple Open & Close")
	}

	if simpleListenerTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"))
		l.SetTimeout(time.Second * 1)
		go func() {
			time.Sleep(time.Millisecond * 100)
			_, _, err = l.WaitOnConnection()
			chk.ErrIs(err, nwk.Err_Timeout)
		}()
		chk.Err(waitFor("Listener Waiting"))
		time.Sleep(time.Second * 2)
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Timeout waiting for connection")
	}

	if simpleListenerTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"))
		go func() {
			time.Sleep(time.Millisecond * 100)
			_, _, err = l.WaitOnConnection()
			chk.ErrIs(err, nwk.Err_NoConnection)
		}()
		chk.Err(waitFor("Listener Waiting"))
		time.Sleep(time.Second)
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		time.Sleep(time.Second) // time for GO routine to exit
		chk.ShowPassFail(t, "Close while waiting")
	}
}

// ------------------------------------------------------------------------- //

func Test_SimpleClientTests(t *testing.T) {
	tst.Testing("Simple Client tests", "", simpleClientTests)

	if simpleClientTests {
		chk.Reset()
		_, err := NewClient(loopback, 0, false)
		chk.ErrIs(err, nwk.Err_ConnectionRefused)
		chk.ShowPassFail(t, "Refused by unopened listener")
	}

	if simpleClientTests {
		chk.Reset()
		_, err := NewClient(loopback, time.Microsecond, false)
		chk.ErrIs(err, nwk.Err_Timeout)
		chk.ShowPassFail(t, "Timeout on listener")
	}
}

// ------------------------------------------------------------------------- //

func Test_SimpleReadTests(t *testing.T) {
	tst.Testing("Simple Server Write, Client Read tests", "", simpleReadTests)

	if simpleReadTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				srw := NewReadWriter(s)
				err = srw.WriteByte(0xAB)
				chk.Err(err, "Write byte failed")
				s.Close()
			}
		}()

		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, false)
		chk.Err(err, "Failed to create client", t.FailNow)
		r, err := crw.ReadByte()
		chk.Tru(r == 0xAB, "ReadByte invalid")
		chk.Err(err, "ReadByte failed: %v", err)
		time.Sleep(time.Millisecond * 100)
		r, err = crw.ReadByte() // read should fail as connection was closed
		chk.ErrIs(err, io.EOF)
		crw.Close()

		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Server Write/ Client Read byte")
	}

	if simpleReadTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				srw := NewReadWriter(s)
				err = srw.Write([]byte("String as bytes"))
				chk.Err(err, "Write []byte failed")
				s.Close()
			}
		}()

		b := make([]byte, 15)
		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, false)
		chk.Err(err, "Failed to create client", t.FailNow)
		r, err := crw.Read(b)
		chk.Tru(r == 15, "Read invalid")
		chk.Tru(string(b) == "String as bytes", "Read invalid")
		chk.Err(err, "Read failed: %v", err)
		time.Sleep(time.Millisecond * 100)
		r, err = crw.Read(b) // read should fail as connection was closed
		chk.ErrIs(err, io.EOF)
		crw.Close()

		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Server Write/ Client Read []byte")
	}

	if simpleReadTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				srw := NewReadWriter(s)
				err = srw.WriteString("Hello, you have reached the test listener\n")
				chk.Err(err, "Write string failed")
				s.Close()
			}
		}()

		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, false)
		chk.Err(err, "Failed to create client", t.FailNow)
		r, err := crw.ReadString() // read the 'hello' message
		chk.Tru(r == "Hello, you have reached the test listener\n", "ReadString invalid")
		chk.Err(err, "ReadString failed: %v", err)
		time.Sleep(time.Millisecond * 100)
		r, err = crw.ReadString() // read should fail as connection was closed
		chk.ErrIs(err, io.EOF)
		crw.Close()

		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Server Write/ Client Read string")
	}
}

// ------------------------------------------------------------------------- //

func Test_SimpleReadTimeout(t *testing.T) {
	tst.Testing("Simple Read timeout", "", clientReadTimeout)

	if clientReadTimeout {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				time.Sleep(time.Second * 2)
				s.Close()
			}
		}()

		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, false)
		chk.Err(err, "Failed to create client", t.FailNow)
		crw.ReadTimeout(time.Second)
		_, err = crw.ReadString() // try to read the 'hello' message
		err = nwk.ChkNetErr(err)  // strip err down to essential err
		chk.ErrIs(err, nwk.Err_Timeout)
		time.Sleep(time.Millisecond * 100)
		crw.Close()

		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Client Read timeout")
	}
}

// ------------------------------------------------------------------------- //

func Test_ReadConnTests(t *testing.T) {
	var l *Listener
	var err error
	tst.Testing("Do multiple read connections", "", multipleReadConns)

	if multipleReadConns {
		l, err = NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		go func() {
			time.Sleep(time.Millisecond * 100)
			for {
				s, _, err := l.WaitOnConnection()
				if nwk.Err_NoConnection == err {
					break
				}
				if nil != err {
					serrPipe <- err
				}
				srw := NewReadWriter(s)
				srw.WriteString("Hello, you have reached the test listener\n")
				s.Close()
			}
		}()
	}

	if multipleReadConns {
		chk.Reset()
		for i := 0; i < 10 && chk.Ok(); i++ {
			chk.Err(waitFor("Listener Waiting"))
			crw, err := NewClient(loopback, 0, false)
			chk.Err(err, "Failed to create client", t.FailNow)
			r, err := crw.ReadString() // read the 'hello' message
			chk.Tru(r == "Hello, you have reached the test listener\n", "ReadString invalid")
			chk.Err(err, "ReadString failed: %v", err)
			time.Sleep(time.Millisecond * 100)
			r, err = crw.ReadString() // read should fail as connection was closed
			chk.ErrIs(err, io.EOF)
			crw.Close()
		}
		// will get a final Waiting from the server loop
		chk.Err(waitFor("Listener Waiting"))
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Server Write / Client Read strings over multiple connects")
	}
}

// ------------------------------------------------------------------------- //

func Test_SimpleWriteTests(t *testing.T) {
	tst.Testing("Simple Client Write, Server Read tests", "", simpleWriteTests)

	if simpleWriteTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				srw := NewReadWriter(s)
				b, err := srw.ReadByte()
				chk.Tru(b == 0xBA, "ReadByte invalid")
				chk.Err(err, "ReadByte failed: %v", err)
				time.Sleep(time.Millisecond * 100)
				b, err = srw.ReadByte()
				chk.ErrIs(err, io.EOF)
				s.Close()
			}
		}()

		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, false)
		chk.Err(err, "Failed to create client", t.FailNow)
		err = crw.WriteByte(0xBA)
		chk.Err(err, "Write byte failed")
		crw.Close()
		time.Sleep(time.Second)
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Client Write/ Server Read byte")
	}

	if simpleWriteTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				srw := NewReadWriter(s)
				b := make([]byte, 15)
				r, err := srw.Read(b)
				chk.Tru(r == 15, "Read invalid")
				chk.Tru(string(b) == "String as bytes", "Read invalid")
				chk.Err(err, "Read failed: %v", err)
				time.Sleep(time.Millisecond * 100)
				r, err = srw.Read(b) // read should fail as connection was closed
				chk.ErrIs(err, io.EOF)
				s.Close()
			}
		}()

		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, false)
		chk.Err(err, "Failed to create client", t.FailNow)
		err = crw.Write([]byte("String as bytes"))
		chk.Err(err, "Write []byte failed")
		crw.Close()
		time.Sleep(time.Second)
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Client Write/ Server Read []byte")
	}

	if simpleWriteTests {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				srw := NewReadWriter(s)
				r, err := srw.ReadString() // read the 'hello' message
				chk.Tru(r == "Hello, I am the simple test client\n", "ReadString invalid")
				chk.Err(err, "ReadString failed: %v", err)
				time.Sleep(time.Millisecond * 100)
				r, err = srw.ReadString() // read should fail as connection was closed
				chk.ErrIs(err, io.EOF)
				s.Close()
			}
		}()

		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, false)
		chk.Err(err, "Failed to create client", t.FailNow)
		err = crw.WriteString("Hello, I am the simple test client\n")
		chk.Err(err, "Write string failed")
		crw.Close()
		time.Sleep(time.Second)
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Client Write/ Server Read string")
	}
}

// ------------------------------------------------------------------------- //

func Test_SimpleWriteTimeout(t *testing.T) {
	tst.Testing("Simple Write timeout", "", clientWriteTimeout)

	if clientWriteTimeout {
		chk.Reset()
		l, err := NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		time.Sleep(time.Millisecond * 100)
		go func() {
			s, _, err := l.WaitOnConnection()
			if nil != err {
				serrPipe <- err
			} else {
				time.Sleep(time.Second * 2)
				s.Close()
			}
		}()

		chk.Err(waitFor("Listener Waiting"))
		crw, err := NewClient(loopback, 0, true)
		chk.Err(err, "Failed to create client", t.FailNow)
		crw.WriteTimeout(time.Microsecond)
		err = crw.WriteString("This will fail on close")
		err = nwk.ChkNetErr(err) // strip err down to essential err
		chk.Err(err, "Should not get timeout error, right?")
		time.Sleep(time.Millisecond * 100)
		err = crw.Close()
		err = nwk.ChkNetErr(err) // strip err down to essential err
		chk.ErrIs(err, nwk.Err_Timeout)
		time.Sleep(time.Millisecond * 100)

		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Client Write timeout")
	}
}

// ------------------------------------------------------------------------- //

func Test_WriteConnTests(t *testing.T) {
	var l *Listener
	var err error
	tst.Testing("Do multiple write connections", "", multipleWriteConns)

	if multipleWriteConns {
		l, err = NewListener(loopback, tstatPipe)
		chk.Err(err, "Failed to create loopback listener", t.FailNow)
		chk.Err(waitFor("Listener Created"), t.FailNow)
		go func() {
			time.Sleep(time.Millisecond * 100)
			for {
				s, _, err := l.WaitOnConnection()
				if nwk.Err_NoConnection == err {
					break
				}
				if nil != err {
					serrPipe <- err
				}
				srw := NewReadWriter(s)
				r, err := srw.ReadString() // read the 'hello' message
				chk.Tru(r == "Hello, I am the simple test client\n", "ReadString invalid")
				chk.Err(err, "ReadString failed: %v", err)
				r, err = srw.ReadString() // read should fail as connection was closed
				chk.ErrIs(err, io.EOF)
				time.Sleep(time.Millisecond * 100)
				s.Close()
			}
		}()
	}

	if multipleWriteConns {
		chk.Reset()
		for i := 0; i < 10 && chk.Ok(); i++ {
			chk.Err(waitFor("Listener Waiting"))
			crw, err := NewClient(loopback, 0, false)
			chk.Err(err, "Failed to create client", t.FailNow)
			err = crw.WriteString("Hello, I am the simple test client\n")
			chk.Err(err, "Write failed")
			crw.Close()
		}
		// will get a final Waiting from the server loop
		chk.Err(waitFor("Listener Waiting"))
		l.Close()
		chk.Err(waitFor("Listener Closed"))
		chk.ShowPassFail(t, "Client Write / Server Read strings over multiple connects")
	}
}

// ------------------------------------------------------------------------- //

type (
	inputStruct struct {
		U32   [4]uint32
		F64   [3]float64
		Bytes [8]byte
		U64   [6]uint64
		F32   [5]float32
	}
	outputStruct struct {
		F64   [3]float64
		Bytes [8]byte
		U64   [6]uint64
		F32   [5]float32
		U32   [4]uint32
	}
)

func recordConnHandler(connectionNumber int, serving string, rw ReadWriter) error {
	var r string
	var err error
	// set a read timeout to see when client closes
	rw.ReadTimeout(time.Second)
	sstatPipe <- fmt.Sprintf("recordConnHandler started: %d) %s", connectionNumber, serving)
forloop:
	for {
		// read request type from client
		r, err = rw.ReadString()
		if nil != err {
			// serrPipe <- err -- handleConn outputs errs
			break
		}
		switch strings.TrimSpace(r) {
		case "RecChunks":
			err = rw.Write([]byte(stringToRecChunks(story)))
		case "SzChunks":
			r, err = rw.ReadString()
			if nil != err {
				// serrPipe <- err -- handleConn outputs errs
				break forloop
			}
			sz, _ := strconv.Atoi(strings.TrimSpace(r))
			err = rw.Write([]byte(stringToSizedChunks(story, sz)))
		case "Struct":
			var is inputStruct
			var os outputStruct
			var bo binary.ByteOrder

			r, err = rw.ReadString()
			if nil != err {
				// serrPipe <- err -- handleConn outputs errs
				break forloop
			}
			if "0\n" == r {
				bo = binary.BigEndian
			} else {
				bo = binary.LittleEndian
			}

			err = rw.ReadStruct(bo, &is)
			if nil != err {
				break
			}

			for i := 0; i < 4; i++ {
				os.U32[i] = is.U32[i]
			}
			for i := 0; i < 6; i++ {
				os.U64[i] = is.U64[i]
			}
			for i := 0; i < 5; i++ {
				os.F32[i] = is.F32[i]
			}
			for i := 0; i < 3; i++ {
				os.F64[i] = is.F64[i]
			}
			for i := 0; i < 8; i++ {
				os.Bytes[i] = is.Bytes[i]
			}

			err = rw.WriteStruct(bo, os)
		default:
			err = nwk.Err_BadData
		}
		if nil != err {
			// serrPipe <- err -- handleConn outputs errs
			break
		}
	}
	sstatPipe <- "recordConnHandler exiting"
	return err
}

func doRecordRequests(num int) {
	for num > 0 {
		num -= 1
		crw, err := NewClient(loopback, time.Millisecond*100, false)
		chk.Err(err, "Failed to create client")
		if nil != err {
			cerrPipe <- err
			break
		}
		if 0 == (num & 1) {
			crw.ReadTimeout(time.Millisecond * 150)
		}

		err = crw.WriteString("RecChunks\n")
		if nil != err {
			cerrPipe <- err
			crw.Close()
			break
		}
		s := ""
		for nil == err {
			var b []byte
			b, err = crw.ReadRecord([]byte(">>>"), []byte("<<<"))
			if nil == err {
				s += string(b)
			}
		}
		crw.Close()
		// error will be EOF if server closes first, or Timeout if
		//  the readtime is set and expires waiting for server
		if (io.EOF != err) && (nwk.Err_Timeout != err) {
			cerrPipe <- err
		}
		if story != strings.TrimSpace(s) {
			chk.Failed()
			cerrPipe <- nwk.Err_BadData
		}
	}
	tstatPipe <- "Exiting test"
}

func doSizedRecords(num int) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for num > 0 {
		num -= 1
		crw, err := NewClient(loopback, time.Millisecond*100, false)
		chk.Err(err, "Failed to create client")
		if nil != err {
			cerrPipe <- err
			break
		}
		if 0 == (num & 1) {
			crw.ReadTimeout(time.Millisecond * 150)
		}

		sz := rnd.Intn(15) + 4
		err = crw.WriteString(fmt.Sprintf("SzChunks\n%d\n", sz))
		if nil != err {
			cerrPipe <- err
			crw.Close()
			break
		}
		s := ""
		for nil == err {
			var b []byte
			b, err = crw.ReadSizedRecord([]byte(">>>"), sz)
			if nil == err {
				s += string(b)
			}
		}
		crw.Close()
		// error will be EOF if server closes first, or Timeout if
		//  the readtime is set and expires waiting for server
		if (io.EOF != err) && (nwk.Err_Timeout != err) {
			cerrPipe <- err
		}
		if story != strings.TrimSpace(s) {
			chk.Failed()
			cerrPipe <- nwk.Err_BadData
		}
	}
	tstatPipe <- "Exiting test"
}

func doStructRecords(num int) {
	var is inputStruct
	var os outputStruct
	var bo binary.ByteOrder
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// let's send multiple structs in this exchange
	crw, err := NewClient(loopback, time.Millisecond*100, false)
	chk.Err(err, "Failed to create client")
	if nil != err {
		cerrPipe <- err
		return
	}

	for num > 0 {
		num -= 1
		for i := 0; i < 4; i++ {
			is.U32[i] = rnd.Uint32()
		}
		for i := 0; i < 6; i++ {
			is.U64[i] = rnd.Uint64()
		}
		for i := 0; i < 5; i++ {
			is.F32[i] = rnd.Float32()
		}
		for i := 0; i < 3; i++ {
			is.F64[i] = rnd.Float64()
		}
		for i := 0; i < 8; i++ {
			is.Bytes[i] = byte(rnd.Intn(256))
		}
		if 0 == (num & 1) {
			bo = binary.BigEndian
		} else {
			bo = binary.LittleEndian
		}

		err = crw.WriteString(fmt.Sprintf("Struct\n%d\n", num&1))
		if nil != err {
			chk.Failed()
			cerrPipe <- err
			break
		}
		err = crw.WriteStruct(bo, is)
		if nil != err {
			chk.Failed()
			cerrPipe <- err
			break
		}
		err = crw.ReadStruct(bo, &os)
		if nil != err {
			chk.Failed()
			cerrPipe <- err
			break
		}

		badData := false
		for i := 0; i < 4; i++ {
			if is.U32[i] != os.U32[i] {
				badData = true
				break
			}
		}
		for i := 0; i < 6; i++ {
			if is.U64[i] != os.U64[i] {
				badData = true
				break
			}
		}
		for i := 0; i < 5; i++ {
			if is.F32[i] != os.F32[i] {
				badData = true
				break
			}
		}
		for i := 0; i < 3; i++ {
			if is.F64[i] != os.F64[i] {
				badData = true
				break
			}
		}
		for i := 0; i < 8; i++ {
			if is.Bytes[i] != os.Bytes[i] {
				badData = true
				break
			}
		}
		if badData {
			chk.Failed()
			cerrPipe <- nwk.Err_BadData
		}
	}

	crw.Close()
	tstatPipe <- "Exiting test"
}

// Generates random sized chunks of the string with a
//	leading '>>>' and trailing '<<<' for the chunks,
//  with random sized blobs of chars between them
func stringToRecChunks(s string) string {
	out := ""
	rand.Seed(time.Now().UnixNano())

	for 32 < len(s) {
		for c := rand.Intn(15) + 4; c > 0; c-- {
			out += string(rune(rand.Intn(26) + 'a'))
		}
		c := rand.Intn(24) + 4
		out += ">>>" + s[:c] + "<<<"
		s = s[c:]
	}
	for c := rand.Intn(15) + 4; c > 0; c-- {
		out += string(rune(rand.Intn(26) + 'a'))
	}
	out += ">>>" + s + "<<<"
	for c := rand.Intn(15) + 13; c > 0; c-- {
		out += string(rune(rand.Intn(26) + 'a'))
	}

	return out
}

// Generates fixed sized chunks of the string with a
//	leading '>>>' for the chunks, with random sized
//	blobs of chars between them.  The last chunk will
//	be padded with spaces at the end if needed
func stringToSizedChunks(s string, sz int) string {
	out := ""
	rand.Seed(time.Now().UnixNano())

	for sz < len(s) {
		for c := rand.Intn(15) + 4; c > 0; c-- {
			out += string(rune(rand.Intn(26) + 'a'))
		}
		out += ">>>" + s[:sz]
		s = s[sz:]
	}
	for c := rand.Intn(15) + 4; c > 0; c-- {
		out += string(rune(rand.Intn(26) + 'a'))
	}
	out += ">>>" + s + strings.Repeat(" ", sz-len(s))
	for c := rand.Intn(15) + 13; c > 0; c-- {
		out += string(rune(rand.Intn(26) + 'a'))
	}

	return out
}

func Test_Records(t *testing.T) {
	tst.Testing("Testing Sending / Receiving records", "", testRecords)

	piperQuiet = true // disable to see progress, stats & errors
	defer func() { time.Sleep(time.Millisecond * 100); piperQuiet = false }()

	if testRecords {
		chk.Reset()
		l, err := NewListener(loopback, sstatPipe) // use server stat pipe
		chk.Err(err, "Failed to create loopback listener", t.FailNow)

		go doRecordRequests(8)
		go l.HandleRequests(recordConnHandler, serrPipe)

		chk.Err(waitFor("Exiting test"))
		l.Close()
		chk.ShowPassFail(t, "Write/Read records")
	}

	if testRecords {
		chk.Reset()
		time.Sleep(time.Second)
		l, err := NewListener(loopback, sstatPipe) // use server stat pipe
		chk.Err(err, "Failed to create loopback listener", t.FailNow)

		go doSizedRecords(8)
		go l.HandleRequests(recordConnHandler, serrPipe)

		chk.Err(waitFor("Exiting test"))
		l.Close()
		chk.ShowPassFail(t, "Write/Read sized records")
	}

	if testRecords {
		chk.Reset()
		time.Sleep(time.Second)
		l, err := NewListener(loopback, sstatPipe) // use server stat pipe
		chk.Err(err, "Failed to create loopback listener", t.FailNow)

		go doStructRecords(8)
		go l.HandleRequests(recordConnHandler, serrPipe)

		chk.Err(waitFor("Exiting test"))
		l.Close()
		chk.ShowPassFail(t, "Write/Read structs")
	}

}

// ------------------------------------------------------------------------- //

func Test___fini(_ *testing.T) {
	ticker.Stop()
	xitSig <- true
	time.Sleep(time.Second)
}
