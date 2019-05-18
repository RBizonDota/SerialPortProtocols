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
func testHandleConnection(conn net.Conn, prefix string, manageChan chan string, cnf *CNF) {
	name := conn.RemoteAddr().String()
	fmt.Printf("%+v connected\n", name)

	defer conn.Close()
	tcpMsg := tcpMessage{
		Type: 1,
		Cnf:  *cnf,
		Data: "",
	}
	str, _ := json.Marshal(tcpMsg)
	conn.Write(str)

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
		}
	}
}
