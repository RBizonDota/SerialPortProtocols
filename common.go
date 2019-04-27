package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
const READTIMEOUT = 20 * time.Second

//TIMEOUT время ожидания
const TIMEOUT = 3 * time.Second

const connInit = "ConnI"

const connEnd = "ConnE"

const transmitInit = "TranI"

const transmitEnd = "TranE"

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
func openPort(self *conn, cnf *CNF) {
	if self.PortStatus != OK {
		c := &serial.Config{Name: cnf.Name, Baud: cnf.Baud}
		s, err := serial.OpenPort(c)
		if err != nil {
			fmt.Println("Couldn't open port with portName=" + cnf.Name + "! " + err.Error())
			self.PortStatus = FAILED
			return
		}
		self.Port = s
		self.PortStatus = OK
	}
	self.Receive = make(chan string, 5)
	self.Send = make(chan string, 5)

	return
}

func closePort(self *conn, cnf *CNF) {
	if self.PortStatus == OK {
		err := self.Port.Close()
		if err != nil {
			fmt.Println("FAIL!\t " + err.Error())
			return
		}
		self.PortStatus = CLOSED
		self.Receive = nil
		self.Send = nil
		self.Port = nil
	}
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
		_, err := port.Write([]byte(text))
		//fmt.Println("Sending...")

		if err != nil {
			res <- FAILED
			fmt.Println("Couldn't send!")
		}
		res <- OK
	}()
	return res
}
func RSTDetector(port *conn, d *int, data string) (string, int) {
	in := make(chan string, 1)
	readMes(port.Port, in)
	for {
		if *d > 4 {
			//panic("")
			fmt.Println("connection broken, didn't get acc")
			//val := <-readMes(port.Port)
			//fmt.Println("OOOOOPS! " + val)
			port.ConnStatus = 2
			return "-1", FAILED
		}
		_ = <-sendMes(port.Port, data)
		//fmt.Println("-> " + data)
		timer := time.NewTimer(TIMEOUT)
		//fmt.Println("DB\t selecting")
		select {
		case <-timer.C:
			fmt.Println("FAIL!\t timeout happend")
			timer.Stop()
			(*d)++
		case val := <-in:
			if !timer.Stop() {
				<-timer.C
			}
			//fmt.Println("Sending")
			//fmt.Println("DB\t RST out")
			//fmt.Println("RST out")
			return val, OK
		}
	}
}
func SyncSend(port *conn, data string, answer bool) (status int) {
	//if port.ConnStatus == OK {
	var d = 0
	var val string
	if answer {
		val, status = RSTDetector(port, &d, data)
	} else {
		status = <-sendMes(port.Port, data)
	}
	if val == ACC {

	} else {
		//передача данных в поток
	}
	//fmt.Println("SyncSend Passed")
	return status
	//}
}

func SyncRead(port *conn, initOnly bool) (val string, status int) {
	//fmt.Println("ProcessInit")
	in := make(chan string, 1)
	timer := time.NewTimer(READTIMEOUT)
	readMes(port.Port, in)
	select {
	case <-timer.C:
		fmt.Println("FAIL!\t timeout happend, RST")
		timer.Stop()
		//разрыв соединения
		port.ConnStatus = NOTCONNECTED
		return "-1", FAILED
	case val := <-in:
		if !timer.Stop() {
			<-timer.C
		}
		if initOnly {
			if val != connInit {
				fmt.Println("OK!\t   message received, val=" + val + " status=" + strconv.Itoa(NOANSWER))
				return val, NOANSWER
			}
		} else if val == ACC {
			fmt.Println("OK!\t   message received, val=" + val + " status=" + strconv.Itoa(NOANSWER))
			return val, NOANSWER
		}
		sendMes(port.Port, ACC)
		fmt.Println("OK!\t   message received, val=" + val + " status=" + strconv.Itoa(OK))
		return val, OK
	}
	return val, OK
}

func syncSignal(port *conn, mu *sync.Mutex) {
	for port.ConnStatus == OK {
		//fmt.Println("syncInit")
		mu.Lock()
		fmt.Println("\t+ Mutex syncSignal")
		status := SyncSend(port, SYNC, true)
		mu.Unlock()
		fmt.Println("\t- Mutex syncSignal")
		if status == OK {
			//	fmt.Println("OK!\t syncSignal passed")
		} else {
			//	fmt.Println("FAIL!\t syncSignal failed")
		}
		time.Sleep(TIMEOUT)
	}
}

/*------------------------------------------------------------------------------*/

func manageHandler(self *conn, mu *sync.Mutex, mycnf *CNF, cnfname string) {
	cnf := mycnf
	//var stopChan = make(chan struct{}, 1)
	for command := range self.ManageStream {
		//fmt.Println("OK!\t manager init")
		switch command {
		case "ConnInit":
			status := connectInitMaster(self, mu)
			if status == OK {
				fmt.Println("OK!\t   connection made")
			} else if status == FAILED {
				fmt.Println("FAIL!\t connection failed")
			}
		case "Open":
			openPort(self, cnf)
			if self.PortStatus == OK {
				fmt.Println("OK!\t   Port MASTER opened")
			} else {
				fmt.Println("FAIL!\t UNABLE TO OPEN PORT MASTER!!!")
			}
			/*go func() {
				for val := range self.Receive {
					fmt.Println("   <-" + val)
				}
			}()*/
		case "OpenSlave": //Потом убрать, подгрузку названия проводить из конфига
			openPort(self, cnf)
			if self.PortStatus == OK {
				fmt.Println("OK!\t   Port SLAVE opened")
			} else {
				fmt.Println("FAIL!\t UNABLE TO OPEN PORT SLAVE!!!")
			}
			go connectInitSlave(self, mu)
		case "transmitInit":
			transmitDataMaster(self, mu)
		case "ConnEnd":
			connectEndMaster(self, mu)
		case "SetCNF":
			//сериализованный CNF
			data := <-self.ManageStream
			res := &CNF{}
			err := json.Unmarshal([]byte(data), &res)
			if err != nil {
				fmt.Println("FAIL!\t Cannot unmarshal cnf data")
			} else {
				setCnf(res, cnfname)
				cnf = res
			}
		case "Close":
			closePort(self, cnf)
			if self.PortStatus == CLOSED {
				fmt.Println("OK!\t   Port closed")
			} else {
				fmt.Println("FAIL!\t UNABLE TO CLOSE PORT!!!")
			}
		default:
			fmt.Println("FAIL!\t Unknown command!")
		}
	}

}
func connectInitMaster(self *conn, mu *sync.Mutex) int {
	mu.Lock()
	fmt.Println("\t+ Mutex connectInitMaster")
	status := SyncSend(self, connInit, true)
	mu.Unlock()
	fmt.Println("\t- Mutex connectInitMaster")
	if status == OK {
		mu.Lock()
		fmt.Println("\t+ Mutex connectInitMaster 2")
		SyncSend(self, ACC, false)
		mu.Unlock()
		fmt.Println("\t- Mutex connectInitMaster 2")
	} else {
		return FAILED
	}
	self.ConnStatus = OK
	go func() { //Синхросигналы
		//	fmt.Println("OK!\t Process init")
		syncSignal(self, mu)
	}()
	go func() { //Отправка данных (синхронный)
		for self.ConnStatus == OK {
			val := <-self.Send
			mu.Lock()
			fmt.Println("\t+ Mutex master sender")
			SyncSend(self, val, true)
			mu.Unlock()
			fmt.Println("\t- Mutex master sender")
		}
	}()

	return OK
}
func connectInitSlave(self *conn, mu *sync.Mutex) {
	mu.Lock()
	fmt.Println("\t+ Mutex connectInitSlave")
	val, status := SyncRead(self, false)
	mu.Unlock()
	fmt.Println("\t- Mutex connectInitSlave")
	if status != OK {
		fmt.Println("FAIL!\t Something wrong in connectInitSlave, read status=" + strconv.Itoa(status))
		return
	}
	if val == connInit {
		mu.Lock()
		fmt.Println("\t+ Mutex connectInitSlave 2")
		val, status2 := SyncRead(self, false)
		mu.Unlock()
		fmt.Println("\t- Mutex connectInitSlave 2")
		if status2 != NOANSWER {
			fmt.Println("FAIL!\t Something wrong in connectInitSlave, read 2 status=" + strconv.Itoa(status2))
			return
		}
		if val == ACC {
			self.ConnStatus = OK
			fmt.Println("OK!\t connection made")
			go func() { //Прием данных
				for self.ConnStatus == OK {
					mu.Lock()
					fmt.Println("\t+ Mutex slave reader")
					val, _ := SyncRead(self, false)
					mu.Unlock()
					fmt.Println("\t- Mutex slave reader")
					if val == transmitInit {
						transmitDataSlave(self, mu)
					}
					if val == connEnd {
						connectEndSlave(self, mu)
					}
				}
			}()
			go func() {
				for val := range self.Receive {
					fmt.Println("<- " + val)
				}
			}()
		}
	}
}

func connectEndMaster(self *conn, mu *sync.Mutex) {
	mu.Lock()
	fmt.Println("\t+ Mutex connectEndMaster")
	status := SyncSend(self, connEnd, true)
	if status == OK {
		val, _ := SyncRead(self, false)
		if val == connEnd {
			self.ConnStatus = NOTCONNECTED
			fmt.Println("OK!\t Connection succesfully broken")
		}
	}
	mu.Unlock()
	fmt.Println("\t- Mutex connectEndMaster")
}
func connectEndSlave(self *conn, mu *sync.Mutex) {
	mu.Lock()
	fmt.Println("\t+ Mutex connectEndSlave")
	status := SyncSend(self, connEnd, true)
	if status == OK {
		self.ConnStatus = NOTCONNECTED
		fmt.Println("OK!\t Connection succesfully broken")
	}
	mu.Unlock()
	fmt.Println("\t- Mutex connectEndSlave")

	go connectInitSlave(self, mu)
}

func transmitDataMaster(self *conn, mu *sync.Mutex) {
	mu.Lock()
	fmt.Println("\t+ Mutex transmitDataMaster")
	status := SyncSend(self, transmitInit, true)
	if status == OK {
		for {
			val, _ := SyncRead(self, false)
			if val == transmitEnd {
				break
			}
			self.Receive <- val
		}
	}
	mu.Unlock()
	fmt.Println("\t- Mutex transmitDataMaster")

}

func transmitDataSlave(self *conn, mu *sync.Mutex) {
	mu.Lock()
	fmt.Println("\t+ Mutex transmitDataSlave")
	for {
		val := <-self.Send
		SyncSend(self, val, true) //TODO анализировать статус и регистрировать его в канальном уровне
		if val == transmitEnd {
			break
		}
	}
	mu.Unlock()
	fmt.Println("\t- Mutex transmitDataSlave")
}

/*------------------------------------------------------------------------------*/

func CLIParser(stream chan string) {
	var res string
	for {
		fmt.Print("command: ")
		fmt.Scanln(&res)
		if res == "OUT" {
			break
		}
		stream <- res
		fmt.Println("OK!\t command wrote ")
	}
}

type CNF struct {
	Name string
	Baud int
}

func getCnf(cnfname string) *CNF {
	fileInfo, err := os.Stat(cnfname)

	file, err := os.Open(cnfname)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, int(fileInfo.Size()))
	_, err = file.Read(buf)
	if err != nil {
		panic(err)
	}
	res := &CNF{}
	err = json.Unmarshal(buf, &res)
	if err != nil {
		panic(err)
	}
	return res
}
func setCnf(data *CNF, cnfname string) {
	// file, err := os.Open("cnf.json")
	file, err := os.OpenFile(cnfname, os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	err = file.Truncate(0)
	if err != nil {
		panic(err)
	}
	buf, err := json.Marshal(*data)
	if err != nil {
		panic(err)
	}
	_, err = file.Write(buf)
	if err != nil {
		panic(err)
	}
}
