package tcp

import (
	"encoding/binary"
	"time"
)

/*
	Some simple routines to handle TCP communications:

		Creating a simple server...
			l = NewListener()
			while running:
				s = l.WaitOnConnection()
					(do all server actions through the net.Conn)
				s.Close()
				  -- or --
					rw = NewReadWriter(s)
					(do server actions through the ReadWriter for simple I/O)
					rw.Close()	// or s.Close()
			  -- or --
			  	l.HandleARequest(...)
			  		(all actions handled via the ConnHandler)
			  -- or --
			  	go l.HandleRequests(...)
			  		(all actions handled via the ConnHandler)
			l.Close()

		Creating a simple client...
			c = NewClient(ipaddr)
			(do client actions through the ReadWriter for simple I/O)
			c.Close()


	type ConnHandler( int, string, ReadWriter ) error:
		On connection, this func is called with:
			connection number (ref only)
			ip being serviced (ip:port)
			ReadWriter attached to the connection
		ConnHandler should exit with any error received through ReadWriter

	interface ReadWriter:
		ReadWriter.Close() error:
			Closes the network connection

		ReadWriter.SetEOL( byte ):
			Set the EOL byte for ReadBytes, defaults to '\n' - same as ReadString

		ReadWriter.ReadTimeout( time.Duration )
			Sets the timeout duration for read calls -- 0 is no expiry

		ReadWriter.WriteTimeout( time.Duration )
			Sets the timeout duration for write calls -- 0 is no expiry

		ReadWriter.FindStart( []byte ) error:
			Reads data until given stRec is matched, data read is discarded

		ReadWriter.Read( []byte ) ( int, error ):
			Reads data until given 'buf' is full or an error is received (EOF)

		ReadWriter.ReadByte() ( byte, error ):
			Read a single byte or error received

		ReadWriter.ReadBytes() ( []byte, error ):
			Reads data until the EOL byte -- defaults to '\n', change with SetEOL

		ReadWriter.ReadString() ( string, error ):
			Reads data until '\n' byte and returns it as a string

		ReadWriter.ReadRecord( []byte, []byte ) ( []byte, error ):
			Reads a variable sized 'record' of data.  Data can have an optional
			starting sequence, but and ending sequence must be given.
			Data until stRec (if given) is discarded.

		ReadWriter.ReadSizedRecord( []byte, int ) ( []byte, error ):
			Reads a fixed sized 'record' of data.  Data can have an optional
			starting sequence to signal start of data.  Data until stRec
			(if given) is discarded.

		ReadWriter.ReadStruct( binary.ByteOrder, interface{} ) error:
			Reads data assumed to be of the given ByteOrder
			and use it to fill the interface

		ReadWriter.Write( []byte ) error:
			Writes a slice of bytes

		ReadWriter.WriteByte( byte ) error:
			Writes a single byte

		ReadWriter.WriteString( string ) error:
			Writes a string

		ReadWriter.WriteStruct( binary.ByteOrder, interface{} ) error:
			Encodes an interface of the given ByteOrder and sends it
*/

type (
	ConnHandler func(connectionNumber int, serving string, rw ReadWriter) error

	ReadWriter interface {
		Close() error
		SetEOL(eol byte)
		ReadTimeout(to time.Duration)
		WriteTimeout(to time.Duration)

		FindStart(stRec []byte) error
		Read(buf []byte) (int, error)
		ReadByte() (byte, error)
		ReadBytes() ([]byte, error)
		ReadString() (string, error)
		ReadRecord(stRec, enRec []byte) ([]byte, error)
		ReadSizedRecord(stRec []byte, recLen int) ([]byte, error)
		ReadStruct(ord binary.ByteOrder, i interface{}) error
		Flush() error
		Write(dta []byte) error
		WriteByte(byt byte) error
		WriteString(str string) error
		WriteStruct(ord binary.ByteOrder, i interface{}) error
	}
)
