package lsep

// The Socket interface type provides a generic adapter to various IO methods
// regardless if they use raw IP, TCP, UDP or IPC.
// A simple socket only assumes Writing, Reading and Closing a connection, it is not
// concerned with dialing or listening the actual connection.
type Socket interface {

	//Close Socket
	Close() error

	//Write data to connection
	//Blocks until all data is written or if  another write is processing
	//A write error closes the connection as there is no way to ensure that
	//the other peer will be able to recover the connection
	Write([]byte) error

	//Read data from connection
	//Blocks until all data is read
	//A read error closes the connection as there is no way to ensure that
	//the other peer will be able to recover the connection
	Read() ([]byte, error)
}

type ListeningSocket interface {
	Close() error
	Listen(port int) error
	Accept() *Socket
}

type DialingSocket interface {
	Close() error
	Dial(string) error
}
