package lsep

import (
	"fmt"
	"net"
	"sync"
)

var (
	// Internal Buffer size, if smaller messages are used,
	// consider decreasing buffer to lower memory footprint,
	// if speed is important, increase buffer size.
	// Buffer size is in KiB
	InternalReadBufferSize = 1024
)

// This Socket provides a Byte-Slice Socket, not a streaming socket, for TCP-Connections
// The socket is not multi-threaded as LSEP requires exclusive usage of the socket
// while encoding or decoding messages. If you absolutely need multiple streams
// consider opening two seperate connections
type TCPConnectionSocket struct {
	connection *net.TCPConn
	// Only one stream can exclusively occupy the connection at a time
	// so we lock it with this  mutex
	syncLock sync.Mutex
	buffer 	[]byte
	loop chan []byte
	connbuf []byte
	databuf []byte
}

func (t *TCPConnectionSocket) initSocket() {
	t.buffer = nil
	t.loop = make(chan []byte, 1)
	t.connbuf = make([]byte, InternalReadBufferSize * 1024)
	t.databuf = make([]byte, InternalReadBufferSize * 1024)
}

// This provides a Byte-Slice Listener Socket, similar to TCPConnectionSocket
type TCPListenerSocket struct {
	listener *net.TCPListener
	syncLock sync.Mutex
}

// Creates a new listener on the given <listenon> address. The format is the exact same as
// with net.Listen
func NewTCPListener(listenon string) (*TCPListenerSocket, error) {
	t := &TCPListenerSocket{}

	if err := t.Listen(listenon); err != nil {
		return nil, err
	}

	return t, nil
}

// Dials to a remote hosts. The created remote socket will now only read
// whole LSEP messages
func NewTCPConnection(address string) (*TCPConnectionSocket, error) {
	t := &TCPConnectionSocket{}

	if err := t.Dial(address); err != nil {
		return nil, err
	}

	return t, nil
}

// Listen on TCP connection, returns no error if listen was successfull
// listenon is in the format <ip>:<port> or :<port>, just like in net.Listen
func (t *TCPListenerSocket) Listen(listenon string) error {
	t.syncLock.Lock()
	defer t.syncLock.Unlock()

	addr, err := net.ResolveTCPAddr("tcp", listenon)

	if err != nil {
		return fmt.Errorf("Resolving: %s", err)
	}

	listener, err := net.ListenTCP("tcp4", addr)

	if err != nil {
		return fmt.Errorf("Listening: %s", err)
	}

	if listener == nil {
		return fmt.Errorf(`No listener created ¯\_(ツ)_/¯`)
	}

	t.listener = listener

	return nil
}

// Accept an incoming LSEP connection, blocks until a client connects and returns
// a Socket, which can be used as if the client was dialed, or an error.
func (t *TCPListenerSocket) Accept() (Socket, error) {
	t.syncLock.Lock()
	defer t.syncLock.Unlock()

	conn, err := t.listener.AcceptTCP()

	if err != nil {
		return nil, err
	}

	newSocket := &TCPConnectionSocket{
		connection: conn,
		syncLock:   sync.Mutex{},
	}

	newSocket.initSocket()

	return newSocket, nil
}

// Connect to a remote host supporting LSEP
func (t *TCPConnectionSocket) Dial(address string) error {
	t.syncLock.Lock()
	defer t.syncLock.Unlock()

	conn, err := net.Dial("tcp", address)

	if err != nil {
		return err
	}

	t.connection = (conn.(*net.TCPConn))
	t.initSocket()

	return nil
}

// Close connection
func (t *TCPConnectionSocket) Close() error {
	t.syncLock.Lock()
	defer t.syncLock.Unlock()
	return t.connection.Close()
}

// Write creates a frame, writes it on wire and then sends all data at once.
// Usually this means the client will receive a tiny packet with the frame
// first and then the data, but some systems may mix the frame with the
// data.
// Blocks until all data is written or returns an error, which closes the connection
func (t *TCPConnectionSocket) Write(data []byte) error {
	t.syncLock.Lock()
	defer t.syncLock.Unlock()

	f := CreateFrame(HeaderVersion1, HeaderOptionRaw, uint64(len(data)))

	n, err := t.connection.Write(f.GetFrame())
	if n != len(f.GetFrame()) {
		t.connection.Close()
		return fmt.Errorf("Frame: Wanted to write %d bytes but wrote %d", len(f.GetFrame()), n)
	}
	if err != nil {
		t.connection.Close()
		return err
	}
	n, err = t.connection.Write(data)
	if n != len(data) {
		t.connection.Close()
		return fmt.Errorf("Frame: Wanted to write %d bytes but wrote %d", len(data), n)
	}
	if err != nil {
		t.connection.Close()
		return err
	}

	return nil
}

func (t *TCPConnectionSocket) clearBuffers() {
	t.initSocket()
}

func (t *TCPConnectionSocket) readExactBytes(size int) ([]byte, error) {
	if size > InternalReadBufferSize * 1024 {
		return nil, fmt.Errorf("Attempted to exceed buffer by %d bytes", size - (InternalReadBufferSize * 1024))
	}

	// TODO: Kill this line somehow, I don't know why this is needed
	t.databuf = make([]byte, InternalReadBufferSize * 1024)

	remLen := size

	if t.buffer != nil && len(t.buffer) > 0 {
		if len(t.buffer) < size {
			copy(t.databuf[:len(t.buffer)], t.buffer)
			remLen -= len(t.buffer)
			t.buffer = nil
		} else if len(t.buffer) == size {
			copy(t.databuf[:size],t.buffer)
			t.buffer = nil
			return t.databuf[:size], nil
		} else if len(t.buffer) > size {
			copy(t.databuf[:size], t.buffer[size:])
			t.buffer = t.buffer[size:]
			return t.databuf[:size], nil
		}
	}
	for remLen > 0 {
		n, err := t.connection.Read(t.connbuf[:size])
		if n > 0 {
			if err != nil {
				t.connection.Close()
				return nil, fmt.Errorf("Fatal: %s", err)
			}

			pos := n
			if pos > remLen {
				pos = remLen
			}

			copy(t.databuf[size - remLen:size - remLen + pos], t.connbuf[:pos])
			t.buffer = t.connbuf[pos:n]
			remLen -= n
		}
	}
	return t.databuf[:size], nil
}

// Read will read 1 LSEP message from wire and return it.
// It will block until the full message was received.
// If read returns an error, the connection must be reestablished
// as a defect will automatically close the connection for safety.
func (t *TCPConnectionSocket) Read() ([]byte, error) {
	// Take socket with exclusive read
	t.syncLock.Lock()
	// Unlock when we're done
	defer t.syncLock.Unlock()

	t.clearBuffers()

	// Reads at minimum 2 bytes from the buffer/connection
	frame, err := t.readExactBytes(2)

	// If there was an error while reading from the connection we might
	// as well close it
	if err != nil {
		t.connection.Close()
		return nil, err
	}

	// Try to parse the frame
	f, f_data, err := ParseFrame(frame)
	if err != nil && f_data == nil {
		//There is no try, do or do not
		t.connection.Close()
		return nil, err
	} else if err != nil && f_data != nil {
		additional_frame, err := t.readExactBytes(len(f_data))
		if err != nil {
			t.connection.Close()
			return nil, err
		}
		frame = append(frame, additional_frame...)

		f, f_data, err = ParseFrame(frame)
		if err != nil {

			fmt.Printf("Error on: %#x\n", frame)
			t.connection.Close()
			return nil, err
		}
	}

	var data = make([]byte, f.Length)
	copy(data[:len(f_data)], f_data)

	// Check that we aren't using any options and the header is version1
	if f.Header.Options != HeaderOptionRaw {
		return nil, fmt.Errorf("Invalid Options for Frame")
	}
	if f.Header.Version != HeaderVersion1 {
		return nil, fmt.Errorf("Invalid Version in Header")
	}

	// Calculate how much data we have missing, if any
	remLen := int(f.Length) - len(f_data)

	// If we need more data, enter stage two
	for remLen > 0 {
		// Use a buffer of the configured internal size, usually 1MiB
		buffSize := InternalReadBufferSize * 1024
		// If the remaining data is smaller than the buffer size, use
		// a smaller buffer
		if buffSize > remLen {
			buffSize = remLen
		}
		// Read missing data blocks from the wire
		newDat, err := t.readExactBytes(buffSize)
		if err != nil {
			t.connection.Close()
			return nil, err
		}

		// Append data and decrement remaining data counter
		copy(data[len(data) - remLen:len(data) - remLen + len(newDat)], newDat)
		remLen -= len(newDat)
	}

	return data, nil
}
