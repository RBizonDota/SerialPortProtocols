package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

func testHandleConnection(conn net.Conn, prefix string, manageChan chan string) {
	data := &tcpMessage{
		Type: 0,
		Cnf:  CNF{},
		Data: "mydata",
	}
	test, _ := json.Marshal(data)
	name := conn.RemoteAddr().String()

	fmt.Printf("%+v connected\n", name)
	conn.Write(test)

	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "Exit" {
			//Для выхода из цикла, в рамках общения не используется
			fmt.Println(prefix, name, "disconnected")
			break
		} else if text != "" {
			fmt.Println(prefix, name, "enters", text)
			mydata := tcpMessage{}
			err := json.Unmarshal([]byte(text), &mydata)
			if err != nil {
				fmt.Println(prefix, name, "error while parsing tcpMessage")
			} else {
				fmt.Println(prefix, name, "OK parsing tcpMessage")
				switch mydata.Type {
				case 0:
					manageChan <- mydata.Data
				case 1:
					manageChan <- "SetCNF"
					str, _ := json.Marshal(mydata.Cnf)
					manageChan <- string(str)
					//TODO нет обработки асинхронной отправки пакетов
				}
			}
			//conn.Write([]byte("You enter " + text + "\n\r"))
		}
	}
}
func slaveListener(manageChan chan string) {
	listner, err := net.Listen("tcp", ":8889")
	fmt.Println("listener 8889 started")
	if err != nil {
		panic(err)
	}
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
		for data := range <-out {
			fmt.Println("Data sent to manage stream, data = ", data)
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
