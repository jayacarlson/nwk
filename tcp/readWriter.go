package tcp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/nwk"
)

/*
	Able to use these routines given a net.Conn (e.g. from WaitOnConnection)

		NewReadWriter( net.Conn ) *ReadWriter:
			Create a new ReadWriter to receive/send data
			All write operations are done directly to the net.Conn,
			while the reads are handled by bufio for buffered reads

		NewReadBufWriter( net.Conn ) *ReadWriter:
			Create a new ReadWriter to receive/send data
			All read & write operations are handled by bufio
			-- Note, user will then need to use Flush() to send any
			writes, or they won't happen until Close()
*/

type (
	readWriter struct {
		srvrIP       string        // if we are a client, this is who we are connected to
		conn         net.Conn      // writer goes direct to net.Conn
		reader       *bufio.Reader // reading is always done buffered
		eol          byte          // EOL byte for ReadBytes
		readTimeout  time.Duration
		writeTimeout time.Duration
	}
	readBufWriter struct {
		r *readWriter   // reading is done through readWriter
		w *bufio.Writer // writing is done buffered using readBufWriter
	}
)

/*
	Create new ReadWriter -- buffered reads only
*/
func NewReadWriter(conn net.Conn) ReadWriter {
	return newReadWriter(conn)
}

/*
	Create new ReadWriter -- buffered reads and writes
*/
func NewReadBufWriter(conn net.Conn) ReadWriter {
	return newReadBufWriter(conn)
}

// ========================================================================= //

func (x *readWriter) Close() error {
	err := nwk.ChkNetErr(x.conn.Close())
	if "" != x.srvrIP {
		ClientDbg.Info("Connection to %s closed (%v)", x.srvrIP, nwk.ChkNetErr(err))
	}
	return err
}

func (x *readWriter) SetEOL(eol byte) {
	x.eol = eol
}

func (x *readWriter) ReadTimeout(to time.Duration) {
	x.readTimeout = to
}

func (x *readWriter) WriteTimeout(to time.Duration) {
	x.writeTimeout = to
}

// ========================================================================= //

func (x *readWriter) FindStart(stRec []byte) error {
	x.setRExpiry()
waitStart:
	for i := 0; i < len(stRec); i++ {
		b, err := x.ReadByte()
		if nil != err {
			return err
		}
		if b != stRec[i] {
			goto waitStart
		}
	}
	return nil
}

func (x *readWriter) Read(buf []byte) (int, error) {
	x.setRExpiry()
	n, err := x.reader.Read(buf)
	return n, nwk.ChkNetErr(err)
}

func (x *readWriter) ReadByte() (byte, error) {
	x.setRExpiry()
	b, err := x.reader.ReadByte()
	return b, nwk.ChkNetErr(err)
}

func (x *readWriter) ReadBytes() ([]byte, error) {
	x.setRExpiry()
	b, err := x.reader.ReadBytes(x.eol)
	return b, nwk.ChkNetErr(err)
}

func (x *readWriter) ReadString() (string, error) {
	x.setRExpiry()
	s, err := x.reader.ReadString('\n')
	return s, nwk.ChkNetErr(err)
}

func (x *readWriter) ReadRecord(stRec, enRec []byte) ([]byte, error) {
	dbg.ChkTruX(0 != len(enRec), "Must have enRec mark") // or would read forever
	err := x.FindStart(stRec)
	if nil != err {
		return []byte{}, nwk.ChkNetErr(err)
	}
	data := []byte{}
waitEnd:
	for i := 0; i < len(enRec); i++ {
		b, err := x.ReadByte()
		if nil != err {
			return data, nwk.ChkNetErr(err)
		}
		if b != enRec[i] {
			data = append(data, enRec[:i]...)
			data = append(data, b)
			goto waitEnd
		}
	}
	return data, nil
}

func (x *readWriter) ReadSizedRecord(stRec []byte, recLen int) ([]byte, error) {
	err := x.FindStart(stRec)
	if nil != err {
		return []byte{}, nwk.ChkNetErr(err)
	}
	data := make([]byte, recLen)
	l, err := x.Read(data)
	return data[:l], nwk.ChkNetErr(err)
}

func (x *readWriter) ReadStruct(ord binary.ByteOrder, i interface{}) error {
	bsz := binary.Size(i)
	if bsz == 0 {
		return nil
	}
	data := make([]byte, bsz)
	rsz, err := x.Read(data)
	if nil != err {
		return nwk.ChkNetErr(err)
	}
	if rsz != bsz {
		return nwk.Err_BadData
	}
	return binary.Read(bytes.NewBuffer(data), ord, i)
}

// ========================================================================= //

func (x *readWriter) Flush() error { return nil }

func (x *readWriter) Write(dta []byte) error {
	x.setWExpiry()
	_, err := x.conn.Write(dta)
	return nwk.ChkNetErr(err)
}

func (x *readWriter) WriteByte(byt byte) error {
	x.setWExpiry()
	dta := []byte{byt}
	_, err := x.conn.Write(dta)
	return nwk.ChkNetErr(err)
}

func (x *readWriter) WriteString(str string) error {
	x.setWExpiry()
	_, err := x.conn.Write([]byte(str))
	return nwk.ChkNetErr(err)
}

func (x *readWriter) WriteStruct(ord binary.ByteOrder, i interface{}) error {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, ord, i)
	if nil != err { // dbg.ChkErr(err, "binary.Write failed: %v", err)
		return err
	}
	return x.Write(buf.Bytes())
}

// ========================================================================= //

func (x *readBufWriter) Close() error {
	ferr := x.w.Flush()
	cerr := x.r.Close()
	if nil != ferr {
		return ferr
	}
	return cerr
}

func (x *readBufWriter) SetEOL(eol byte) {
	x.r.eol = eol
}

func (x *readBufWriter) ReadTimeout(to time.Duration) {
	x.r.readTimeout = to
}

func (x *readBufWriter) WriteTimeout(to time.Duration) {
	x.r.writeTimeout = to
}

// ========================================================================= //

func (x *readBufWriter) FindStart(stRec []byte) error { return x.r.FindStart(stRec) }
func (x *readBufWriter) Read(buf []byte) (int, error) { return x.r.Read(buf) }
func (x *readBufWriter) ReadByte() (byte, error)      { return x.r.ReadByte() }
func (x *readBufWriter) ReadBytes() ([]byte, error)   { return x.r.ReadBytes() }
func (x *readBufWriter) ReadString() (string, error)  { return x.r.ReadString() }
func (x *readBufWriter) ReadRecord(stRec, enRec []byte) ([]byte, error) {
	return x.r.ReadRecord(stRec, enRec)
}
func (x *readBufWriter) ReadSizedRecord(stRec []byte, recLen int) ([]byte, error) {
	return x.r.ReadSizedRecord(stRec, recLen)
}
func (x *readBufWriter) ReadStruct(ord binary.ByteOrder, i interface{}) error {
	return x.r.ReadStruct(ord, i)
}

// ========================================================================= //

func (x *readBufWriter) Flush() error { return x.w.Flush() }

func (x *readBufWriter) Write(dta []byte) error {
	x.r.setWExpiry()
	_, err := x.w.Write(dta)
	return nwk.ChkNetErr(err)
}

func (x *readBufWriter) WriteByte(byt byte) error {
	x.r.setWExpiry()
	dta := []byte{byt}
	_, err := x.w.Write(dta)
	return nwk.ChkNetErr(err)
}

func (x *readBufWriter) WriteString(str string) error {
	x.r.setWExpiry()
	_, err := x.w.Write([]byte(str))
	return nwk.ChkNetErr(err)
}

func (x *readBufWriter) WriteStruct(ord binary.ByteOrder, i interface{}) error {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, ord, i)
	if nil != err { // dbg.ChkErr(err, "binary.Write failed: %v", err)
		return err
	}
	return x.Write(buf.Bytes())
}

// ------------------------------------------------------------------------- //

func newReadWriter(conn net.Conn) *readWriter {
	x := readWriter{
		conn:   conn,
		reader: bufio.NewReader(conn),
		eol:    '\n',
	}
	return &x
}

func newReadBufWriter(conn net.Conn) *readBufWriter {
	x := readBufWriter{
		r: newReadWriter(conn),
		w: bufio.NewWriter(conn),
	}
	return &x
}

func (x *readWriter) setRExpiry() {
	expiry := time.Time{}
	if 0 != x.readTimeout {
		expiry = time.Now().Add(x.readTimeout)
	}
	x.conn.SetReadDeadline(expiry)
}

func (x *readWriter) setWExpiry() {
	expiry := time.Time{}
	if 0 != x.writeTimeout {
		expiry = time.Now().Add(x.writeTimeout)
	}
	x.conn.SetWriteDeadline(expiry)
}
