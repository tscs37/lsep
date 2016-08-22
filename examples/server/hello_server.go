package main

import (
	"bytes"
	"fmt"

	"github.com/tscs37/lsep"
)

func main() {

	var toSend = ""
	var toSendBuffer bytes.Buffer
	fmt.Printf("Generating...")
	for i := 0; i < 500*1024*1024; i++ {
		if i%(200*1024*1024) == 0 {
			fmt.Printf("%d kb...", i/1024)
		}
		toSendBuffer.WriteByte(0xFF)
	}

	toSend = string(toSendBuffer.Bytes())

	fmt.Println("Done!")

	listener, err := lsep.NewTCPListener("127.0.0.1:13338")

	if err != nil {
		fmt.Printf("Error while listening: %s", err)
		return
	}

	for {
		fmt.Printf("Waiting for listeners...\n")
		socket, err := listener.Accept()

		fmt.Printf("New listener!\n")

		if err != nil {
			fmt.Printf("Error while establishing connection: %s", err)
			return
		}

		fmt.Printf("Beginning Data Session...\n")

		hello := []byte("Hello World")

		fmt.Printf("Sending %d bytes\n", len(hello))
		err = socket.Write(hello)

		if err != nil {
			fmt.Printf("Error while sending: %s", err)
			return
		}

		fmt.Printf("Sending Second Part...\n")

		err = socket.Write([]byte(toSend))

		if err != nil {
			fmt.Printf("Error while sending: %s\n", err)
		}

		fmt.Printf("Finished Session, closing...\n")

		err = socket.Close()

		if err != nil {
			fmt.Printf("Error while closing: %s\n", err)
		}
	}
}
