package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

const manageMSG = 1
const sendMSG = 2
const socketOnly = 3

type tcpMessage struct {
	Type int    //Тип сообщения: куда его писать
	Cnf  CNF    //Конфиг, только для установки конфига, во всех остальных случаях nil
	Data string //Данные: если в управляющий поток, то команда, если в выходной, то сообщение

}

//self, cnf
func handleConnection(conn net.Conn, self *conn, mycnf *CNF) {
	name := conn.RemoteAddr().String()

	fmt.Printf("%+v connected\n", name)
	data, err := json.Marshal(&tcpMessage{
		Type: socketOnly,
		Cnf:  *mycnf,
		Data: "cnf",
	})
	if err != nil {
		fmt.Println("FAIL!\t Couldn't marshal tcpMessage")
	}
	conn.Write(data)

	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "OUT" {
			fmt.Println(name, "disconnected")
			break
		} else if text != "" {
			fmt.Println(name, "enters", text)
			msg := &tcpMessage{}
			err := json.Unmarshal([]byte(text), msg)
			if err != nil {
				fmt.Println("FAIL!\t Couldn't unmarshal tcpMessage")
			} else {
				if msg.Type == manageMSG {
					self.ManageStream <- msg.Data
					if msg.Data == "SetCNF" {
						data, err := json.Marshal(msg.Cnf)
						if err != nil {
							fmt.Println("FAIL!\t Couldn't marshal CNF")
						} else {
							self.ManageStream <- string(data)
						}
					}
				} else if msg.Type == sendMSG {
					self.Send <- msg.Data
				} else {
				}
			}
			//conn.Write([]byte("You enter " + text + "\n\r"))
		}
	}
}
