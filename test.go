package main

import (
	"bufio"
	"fmt"
	"net"
)

func testHandleConnection(conn net.Conn, prefix string) {
	name := conn.RemoteAddr().String()

	fmt.Printf("%+v connected\n", name)
	conn.Write([]byte("Hello, " + name + "\n\r"))

	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "Exit" {
			conn.Write([]byte("Bye\n\r"))
			fmt.Println(prefix, name, "disconnected")
			break
		} else if text != "" {
			fmt.Println(prefix, name, "enters", text)
			//conn.Write([]byte("You enter " + text + "\n\r"))
		}
	}
}
func slaveListener() {
	listner, err := net.Listen("tcp", ":8889")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}
		go testHandleConnection(conn, "SLAVE ")
	}
}

func main() {
	listner, err := net.Listen("tcp", ":8888")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listner.Accept()
		if err != nil {
			panic(err)
		}
		go testHandleConnection(conn, "MASTER")
	}
}
