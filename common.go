package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/tarm/serial"
)

//Статусы соединений, портов
/*------------------------------------------------------------------------------*/

const CLOSED = -1

//OK статус успешного открытия соединения, порта
const OK = 0

//FAILED статус проваленного открытия соединения, порта
const FAILED = 1

//NOTCONNECTED статус только открытого порта, либо разорванного соединения
const NOTCONNECTED = 2

const NOANSWER = 4

/*------------------------------------------------------------------------------*/

//PACKETLEN длина кадра
const PACKETLEN = 5

//MASTER название порта мастера
const MASTER = "COM1"

//SLAVE название порта раба
const SLAVE = "COM2"

//TIMEOUT время ожидания
const TIMEOUT = 3 * time.Second

const connInit = "ConnInit"

//ACC сообщение подтверждения
const ACC = "A"

const SYNC = "Q"

const INFO = "I"

/*------------------------------------------------------------------------------*/

type conn = struct {
	Port         *serial.Port
	ConnStatus   int
	PortStatus   int
	ManageStream chan string
	Receive      chan string
	Send         chan string
}

func NewConn() conn {
	stream := make(chan string, 1)
	res := conn{
		Port:         nil,
		ConnStatus:   NOTCONNECTED,
		PortStatus:   CLOSED,
		ManageStream: stream,
		Receive:      nil,
		Send:         nil,
	}
	return res
}
func openPort(self *conn, portName string) {
	c := &serial.Config{Name: portName, Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		fmt.Println("Couldn't open port with portName=" + portName + "! " + err.Error())
		self.PortStatus = FAILED
		return
	}
	self.Receive = make(chan string, 1)
	self.Send = make(chan string, 1)
	self.PortStatus = OK
	self.Port = s
	return
}

/*------------------------------------------------------------------------------*/

func readMes(s *serial.Port, res chan string) {
	go func() {
		//defer fmt.Println("DB\t readMes out")
		buf := make([]byte, PACKETLEN)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Println("Couldn't read!")
			return
		}
		if string(buf[0]) == ACC {
			res <- ACC
			return
		} else if string(buf[0]) == SYNC {
			res <- SYNC
			return
		}
		if n < PACKETLEN {
			n, err = s.Read(buf[n:PACKETLEN])
		}
		res <- string(buf[:PACKETLEN])
	}()
	return
}
func sendMes(port *serial.Port, text string) chan int {
	res := make(chan int, 1)
	go func() {
		n, err := port.Write([]byte(text))
		//fmt.Println("Sending...")

		if err != nil {
			fmt.Println("Couldn't send!")
		}
		res <- n
	}()
	return res
}
func RSTDetector(port *conn, d *int, data string) string {
	in := make(chan string, 2)
	readMes(port.Port, in)
	for {
		if *d > 4 {
			//panic("")
			fmt.Println("connection broken, didn't get acc")
			//val := <-readMes(port.Port)
			//fmt.Println("OOOOOPS! " + val)
			port.ConnStatus = 2
			return ""
		}
		_ = <-sendMes(port.Port, data)
		//fmt.Println("-> " + data)
		timer := time.NewTimer(TIMEOUT)
		//fmt.Println("DB\t selecting")
		select {
		case <-timer.C:
			fmt.Println("!!!\t timeout happend")
			timer.Stop()
			(*d)++
		case val := <-in:
			if !timer.Stop() {
				<-timer.C
			}
			//fmt.Println("DB\t RST out")
			//fmt.Println("RST out")
			return val
		}
	}
}
func SyncSend(port *conn, data string) (status int) {
	//if port.ConnStatus == OK {
	var d = 0
	val := RSTDetector(port, &d, data)
	if val == ACC {
		//fmt.Println("<- " + val)
		//повторная отправка
	} else {
	}
	//fmt.Println("SyncSend Passed")
	return OK
	//}
	return FAILED
}

func SyncRead(port *conn) (val string, status int) {
	in := make(chan string, 1)
	readMes(port.Port, in)
	val = <-in
	//fmt.Println("\t<- " + val)
	if val == ACC {
		return val, NOANSWER
	}
	sendMes(port.Port, ACC)
	//fmt.Println("SyncRead Passed")
	return val, OK
}

func syncSignal(port *conn, mu *sync.Mutex) {
	for port.ConnStatus == OK {
		//fmt.Println("Sending")
		mu.Lock()
		SyncSend(port, SYNC)
		mu.Unlock()
		//fmt.Println("   OK, syncSignal passed")
		time.Sleep(TIMEOUT)
	}
}

/*------------------------------------------------------------------------------*/

func manageHandler(command string, self *conn, mu *sync.Mutex) {
	switch command {
	case "ConnInit":
		connectInitMaster(self, mu)
		/*if self.ConnStatus == OK {
			go func() {
				for self.PortStatus == OK {
					val := <-self.Send
					SyncSend(self, val)
				}
			}()
		}*/
	case "Open":
		openPort(self, MASTER)
		if self.PortStatus == OK {
			fmt.Println("OK!\t Port MASTER opened")
		} else {
			fmt.Println("FAIL!\t UNABLE TO OPEN PORT MASTER!!!")
		}
	case "OpenSlave": //Потом убрать, подгрузку названия проводить из конфига
		openPort(self, SLAVE)
		if self.PortStatus == OK {
			fmt.Println("OK!\t Port SLAVE opened")
		} else {
			fmt.Println("FAIL!\t UNABLE TO OPEN PORT SLAVE!!!")
		}
	}
}
func connectInitMaster(self *conn, mu *sync.Mutex) {
	mu.Lock()
	SyncSend(self, connInit)
	SyncSend(self, ACC)
	mu.Unlock()
	self.ConnStatus = OK
	go syncSignal(self, mu)
}
func connectInitSlave(self *conn, mu *sync.Mutex) {
	mu.Lock()
	val, _ := SyncRead(self)
	mu.Unlock()
	if val == ACC {
		self.ConnStatus = OK
	}
}
