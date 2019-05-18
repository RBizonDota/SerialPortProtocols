package main

import (
	"encoding/json"
	"fmt"
	"net"
)

func slaveListener(manageChan chan string) {
	listner, err := net.Listen("tcp", ":8889")
	if err != nil {
		panic(err)
	}
	fmt.Println("listener 8889 started")
	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}
		go testHandleConnection(conn, "SLAVE ", manageChan)
	}
}

func main() {
	out := make(chan string)
	go func() {
		for val := range out {
			fmt.Println("Data sent to manage stream, data = ", val)
		}
	}()
	data := &tcpMessage{
		Type: 0,
		Cnf:  CNF{},
		Data: "mydata",
	}
	test, _ := json.Marshal(data)
	fmt.Println(string(test))
	go slaveListener(out)
	listner, err := net.Listen("tcp", ":8888")
	fmt.Println("listener 8888 started")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}
		go testHandleConnection(conn, "MASTER", out)
	}
}
