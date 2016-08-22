package main

import (
	"fmt"
	"time"

	"github.com/tscs37/lsep"
)

func main() {
	for i := 0; i < 10; i++ {
		client()
	}
}

func client() {

	fmt.Printf("Dialing localhost...\n")

	socket, err := lsep.NewTCPConnection("127.0.0.1:13338")

	if err != nil {
		fmt.Printf("Error while dialing: %s\n", err)
		return
	}

	fmt.Printf("Reading from Socket\n")

	hello, err := socket.Read()

	if err != nil {
		fmt.Printf("Error while reading: %s\n", err)
		return
	}

	fmt.Printf("Received: %s\n", hello)

	fmt.Printf("Reading from Socket\n")

	t1 := time.Now()

	test, err := socket.Read()

	t2 := time.Now().Sub(t1)

	if err != nil {
		fmt.Printf("Error while reading: %s\n", err)
		return
	}

	fmt.Printf("Received: %d bytes\n", len(test))
	fmt.Printf("Took    : %s\n", t2)
	fmt.Printf("Speed   : %f GiBps\n", float64(len(test))/(t2.Seconds()*1024*1024*1024))

	socket.Close()
}
