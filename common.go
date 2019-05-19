package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/bits"
	"net"
	"os"
	"strconv"
	"strings"
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
const PACKETLEN = 8

//TIMEOUT время ожидания
const READTIMEOUT = 20 * time.Second

//TIMEOUT время ожидания
const TIMEOUT = 3 * time.Second

const connInitMsg = "ConnIn"

const connEndMsg = "ConnEn"

const transmitInit = "TranIn"

const transmitEnd = "TranEn"

//ACC сообщение подтверждения
const ACC = "AAAAAA"

const SYNC = "QQQQQQ"

const INFO = "I"

/*------------------------------------------------------------------------------*/

type counter = struct {
	CurName      uint16
	NameCadrSize uint16
	CurFile      uint32
	FileCadrSize uint32
	FileName     string
}

type conn = struct {
	Port         *serial.Port
	ConnStatus   int
	PortStatus   int
	ManageStream chan string
	Receive      chan string
	Send         chan string
	callBack     *net.Conn
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
func RSTDetector(port *conn, d *int, mydata string) (string, int) {
	in := make(chan string, 1)
	readMes(port.Port, in)
	for {
		if *d > 4 {
			fmt.Println("connection broken, didn't get acc")
			tcpMsg := tcpMessage{
				Type: 0,
				Cnf:  CNF{},
				Data: "RST",
			}
			str, _ := json.Marshal(tcpMsg)
			(*port.callBack).Write(str)
			port.ConnStatus = 2
			return "-1", FAILED
		}
		_ = <-sendMes(port.Port, mydata)
		//fmt.Println("-> " + mydata)
		timer := time.NewTimer(TIMEOUT)
		//fmt.Println("DB\t selecting")
		select {
		case <-timer.C:
			fmt.Println("FAIL!\t timeout happend")
			timer.Stop()
			tcpMsg := tcpMessage{
				Type: 0,
				Cnf:  CNF{},
				Data: "packetLoss",
			}
			str, _ := json.Marshal(tcpMsg)
			(*port.callBack).Write(str)
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
func SyncSend(port *conn, mydata string, tp string, answer bool) (status int) {
	//if port.ConnStatus == OK {
	nameSlice := AddFrameType([]byte(mydata), tp)
	dataInBits := ToBits(nameSlice)
	data := Code(dataInBits, bits.Len(uint(dataInBits)))
	msgBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(msgBytes, uint64(data))

	var d = 0
	//var val string
	if answer {
		_, status = RSTDetector(port, &d, string(msgBytes))
	} else {
		status = <-sendMes(port.Port, string(msgBytes))
	}
	//fmt.Println("SyncSend Passed")
	return status
	//}
}

func SyncRead(port *conn, initOnly bool) (val string, tp byte, status int) {
	//fmt.Println("ProcessInit")

	in := make(chan string, 1)
	timer := time.NewTimer(READTIMEOUT)
	readMes(port.Port, in)
	select {
	case <-timer.C:
		fmt.Println("FAIL!\t timeout happend, RST")
		timer.Stop()
		//разрыв соединения
		tcpMsg := tcpMessage{
			Type: 0,
			Cnf:  CNF{},
			Data: "RST",
		}
		str, _ := json.Marshal(tcpMsg)
		(*port.callBack).Write(str)

		port.ConnStatus = NOTCONNECTED
		return "-1", 0, FAILED
	case val := <-in:
		if !timer.Stop() {
			<-timer.C
		}

		data := binary.LittleEndian.Uint64([]byte(val))
		decoded, tp, valid := Decode(int64(data), bits.Len(uint(56)))
		if !valid {
			return "", 0, FAILED
		}
		if initOnly {
			if tp != connInit {
				//fmt.Println("OK!\t   message received, val=" + string(decoded) + " status=" + strconv.Itoa(NOANSWER) + " tp = " + strconv.Itoa(int(tp)))
				return string(decoded), tp, NOANSWER
			}
		} else if tp == accCadr {
			//fmt.Println("OK!\t   message received, val=" + string(decoded) + " status=" + strconv.Itoa(NOANSWER) + " tp = " + strconv.Itoa(int(tp)))
			return string(decoded), tp, NOANSWER
		}
		sendMes(port.Port, ACC)
		//fmt.Println("OK!\t   message received, val=" + string(decoded) + " status=" + strconv.Itoa(OK) + " tp = " + strconv.Itoa(int(tp)))
		return string(decoded), tp, OK
	}
	return "", 0, FAILED
}

func syncSignal(port *conn, mu *sync.Mutex) {
	for port.ConnStatus == OK {
		//fmt.Println("syncInit")
		mu.Lock()
		fmt.Println("\t+ Mutex syncSignal")
		status := SyncSend(port, SYNC, "sync", true)
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
			go connectInitSlave(self, mu, cnf)
		case "transmitInit":
			transmitDataMaster(self, mu, cnf)
		case "transmitResume":
			transmitResumeMaster(self, mu, cnf)
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
		case "SetFileName":
			data := <-self.ManageStream
			cnf.FileName = data
		case "SetFileDir":
			data := <-self.ManageStream
			cnf.FileDir = data
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
	status := SyncSend(self, connInitMsg, "init", true)
	mu.Unlock()
	fmt.Println("\t- Mutex connectInitMaster")
	if status == OK {
		mu.Lock()
		fmt.Println("\t+ Mutex connectInitMaster 2")
		SyncSend(self, ACC, "acc", false)
		mu.Unlock()
		fmt.Println("\t- Mutex connectInitMaster 2")
	} else {
		return FAILED
	}
	tcpMsg := tcpMessage{
		Type: 0,
		Cnf:  CNF{},
		Data: "ConnectOK",
	}
	str, _ := json.Marshal(tcpMsg)
	(*self.callBack).Write(str)
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
			SyncSend(self, val, "info", true)
			mu.Unlock()
			fmt.Println("\t- Mutex master sender")
		}
	}()

	return OK
}
func connectInitSlave(self *conn, mu *sync.Mutex, cnf *CNF) {
	mu.Lock()
	fmt.Println("\t+ Mutex connectInitSlave")
	_, tp, status := SyncRead(self, false)
	mu.Unlock()
	fmt.Println("\t- Mutex connectInitSlave status = " + strconv.Itoa(status))
	if status != OK {
		tcpMsg := tcpMessage{
			Type: 0,
			Cnf:  CNF{},
			Data: "Close",
		}
		str, _ := json.Marshal(tcpMsg)
		(*self.callBack).Write(str)
		fmt.Println("FAIL!\t Something wrong in connectInitSlave, read status=" + strconv.Itoa(status))
		return
	}
	if tp == connInit {
		mu.Lock()
		fmt.Println("\t+ Mutex connectInitSlave 2")
		_, tp, status2 := SyncRead(self, false)
		mu.Unlock()
		fmt.Println("\t- Mutex connectInitSlave 2")
		if tp != accCadr {
			fmt.Println("FAIL!\t Something wrong in connectInitSlave, read 2 status=" + strconv.Itoa(status2))
			return
		}
		if tp == accCadr {
			self.ConnStatus = OK
			fmt.Println("OK!\t connection made")
			go func() { //Прием данных
				fmt.Println("Reader Start")
				for self.ConnStatus == OK {
					mu.Lock()
					fmt.Println("\t+ Mutex slave reader")
					val, tp, _ := SyncRead(self, false)
					//fmt.Println("val = " + val + ",  tp = " + strconv.Itoa(int(tp)))
					mu.Unlock()
					fmt.Println("\t- Mutex slave reader")
					if tp == transInitCadr {
						transmitDataSlave(self, mu, cnf.FileName)
					} else if tp == connEnd {
						connectEndSlave(self, mu, cnf)
					} else if tp == infoCadr {
						self.Receive <- val
					} else if tp == transResumeCadr {
						valSlice := []byte(val)
						curNameSizeSlice := valSlice[0:2]
						curFileSizeSlice := valSlice[2:6]
						cnf.ResumeCounter.CurName = binary.LittleEndian.Uint16(curNameSizeSlice)
						cnf.ResumeCounter.CurFile = binary.LittleEndian.Uint32(curFileSizeSlice)
						transmitResumeSlave(self, mu, cnf)
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
	status := SyncSend(self, connEndMsg, "end", true)
	if status == OK {
		_, tp, _ := SyncRead(self, false)
		if tp == connEnd {
			self.ConnStatus = NOTCONNECTED
			fmt.Println("OK!\t Connection succesfully broken")
		}
	}
	mu.Unlock()
	fmt.Println("\t- Mutex connectEndMaster")
}
func connectEndSlave(self *conn, mu *sync.Mutex, cnf *CNF) {
	mu.Lock()
	fmt.Println("\t+ Mutex connectEndSlave")
	status := SyncSend(self, connEndMsg, "end", true)
	if status == OK {
		self.ConnStatus = NOTCONNECTED
		fmt.Println("OK!\t Connection succesfully broken")
	}
	mu.Unlock()
	fmt.Println("\t- Mutex connectEndSlave")

	go connectInitSlave(self, mu, cnf)
}

func transmitDataMaster(self *conn, mu *sync.Mutex, cnf *CNF) {
	if cnf.FileDir == "" {
		fmt.Println("ERROR!!!   dirname not set")
		return
	}
	mu.Lock()
	fmt.Println("\t+ Mutex transmitDataMaster")
	defer fmt.Println("\t- Mutex transmitDataMaster")
	defer mu.Unlock()
	status := SyncSend(self, transmitInit, "transinit", true)
	if status == OK {
		val, _, status := SyncRead(self, false)
		if status != OK {
			return
		}
		fileSizeSlice := []byte(val)[0:4]
		nameSizeSlice := []byte(val)[4:6]
		fileCadrSize := binary.LittleEndian.Uint32(fileSizeSlice)
		nameCadrSize := binary.LittleEndian.Uint16(nameSizeSlice)
		cnf.ResumeCounter.NameCadrSize = nameCadrSize
		cnf.ResumeCounter.FileCadrSize = fileCadrSize
		fmt.Printf("fileCadrSize: %d\n", fileCadrSize)
		fmt.Printf("nameCadrSize: %d\n", nameCadrSize)
		close := true
		defer func() { close = false }()
		go func() {
			for close {
				tcpMsg := tcpMessage{
					Type: 1,
					Cnf:  *cnf,
					Data: "",
				}
				str, _ := json.Marshal(tcpMsg)
				(*self.callBack).Write(str)
				time.Sleep(time.Second / 2)
			}
		}()
		for i := 0; i < int(nameCadrSize); i++ {
			val, _, status := SyncRead(self, false)
			if status == OK {
				//-----обработка последней части имени - возможно переделать---
				n := bytes.Index([]byte(val), []byte{0})
				if i == int(nameCadrSize-1) && n != -1 {
					val = string([]byte(val[:n])) //без конечных нулей
				}
				//--------------------------
				cnf.ResumeCounter.FileName += string(val)
				cnf.ResumeCounter.CurName++
			} else {
				return
			}
		}
		CreateFile(cnf.ResumeCounter.FileName, cnf.FileDir)

		//---------------Получение файла------------------------
		var fileTextBytes []byte
		for i := 0; i < int(fileCadrSize); i++ {
			val, _, status := SyncRead(self, false)
			if status == OK {
				//проверка frameType и valid?
				//fileTextBytes = append(fileTextBytes, []byte(val)...)
				fileTextBytes = []byte(val)
				if i == int(fileCadrSize-1) {
					fileTextBytes = delEndZeros([]byte(val))
				}
				dataAdded := AddDataToFile(cnf.ResumeCounter.FileName, fileTextBytes, cnf.FileDir)
				if dataAdded {
					cnf.ResumeCounter.CurFile++
				} else {
					tcpMsg := tcpMessage{
						Type: 0,
						Cnf:  CNF{},
						Data: "TRERROR",
					}
					str, _ := json.Marshal(tcpMsg)
					(*self.callBack).Write(str)
					return
				}
			} else {
				return
			}
		}
		tcpMsg := tcpMessage{
			Type: 0,
			Cnf:  CNF{},
			Data: "transmitOK",
		}
		str, _ := json.Marshal(tcpMsg)
		(*self.callBack).Write(str)
		fmt.Print("ResumeCounter in Master: ")
		fmt.Println(cnf.ResumeCounter)
		cnf.ResumeCounter = counter{} //обнуление
	}

}

func transmitResumeMaster(self *conn, mu *sync.Mutex, cnf *CNF) {
	if cnf.ResumeCounter == (counter{}) {
		fmt.Println("No file in queue, use transmitInit")
		return
	}
	if cnf.FileDir == "" {
		fmt.Println("ERROR!!!   dirname not set")
		return
	}
	mu.Lock()
	fmt.Println("\t+ Mutex transmitResumeMaster")
	defer fmt.Println("\t- Mutex transmitResumeMaster")
	defer mu.Unlock()

	resumeMsgBytes1 := make([]byte, 4)
	resumeMsgBytes2 := make([]byte, 2)
	binary.LittleEndian.PutUint32(resumeMsgBytes1, uint32(cnf.ResumeCounter.CurFile))
	binary.LittleEndian.PutUint16(resumeMsgBytes2, uint16(cnf.ResumeCounter.CurName))
	resumeMsg := append(resumeMsgBytes2, resumeMsgBytes1...)
	status := SyncSend(self, string(resumeMsg), "transresume", true)
	if status == OK {
		nameCadrSize := cnf.ResumeCounter.NameCadrSize
		fileCadrSize := cnf.ResumeCounter.FileCadrSize
		if cnf.ResumeCounter.CurName != nameCadrSize {
			//---------------Дополучение имени-----------------------
			for i := cnf.ResumeCounter.CurName; i < nameCadrSize; i++ {
				val, _, status := SyncRead(self, false)
				if status == OK {
					//проверка frameType и valid?
					//-----обработка последней части имени - возможно переделать---
					n := bytes.Index([]byte(val), []byte{0})
					if (i == nameCadrSize-1) && (n != -1) {
						val = string([]byte(val[:n])) //без конечных нулей
					}
					//--------------------------
					cnf.ResumeCounter.FileName += string(val)
					cnf.ResumeCounter.CurName++
				} else {
					return
				}
			}
			CreateFile(cnf.ResumeCounter.FileName, cnf.FileDir)
		}
		//---------------Дополучение файла------------------------
		var filePartBytes []byte
		for i := cnf.ResumeCounter.CurFile; i < fileCadrSize; i++ {
			val, _, status := SyncRead(self, false)
			if status == OK {
				//проверка frameType и valid?
				filePartBytes = []byte(val)
				if i == fileCadrSize-1 {
					filePartBytes = delEndZeros([]byte(val))
				}
				dataAdded := AddDataToFile(cnf.ResumeCounter.FileName, filePartBytes, cnf.FileDir)
				if dataAdded {
					cnf.ResumeCounter.CurFile++
				}
			} else {
				return
			}
		}
		fmt.Print("ResumeCounter in resumeMaster: ")
		fmt.Println(cnf.ResumeCounter)
		cnf.ResumeCounter = counter{} //обнуление
	}
}

func transmitResumeSlave(self *conn, mu *sync.Mutex, cnf *CNF) {
	mu.Lock()
	fmt.Println("\t+ Mutex transmitResumeSlave")
	start := time.Now()
	var fileSize, nameSize, i, bytesToRead int64
	file, err := os.Open(DataFileName)
	CheckError(err)

	fmt.Printf("Slave startNameCadr " + string(cnf.ResumeCounter.CurName))
	fmt.Printf("Slave startFileCadr " + string(cnf.ResumeCounter.CurFile))

	stat, err := file.Stat()
	CheckError(err)
	fileSize = stat.Size()
	nameSize = int64(len(DataFileName))
	bytesToRead = 6
	//----------------------Передача названия------------------------------------
	nameBytes := []byte(DataFileName)
	for len(nameBytes)%int(bytesToRead) != 0 { //TODO переписать, неэффективно
		nameBytes = append(nameBytes, 0)
	}
	//fmt.Printf("nameBytes: %b\n", nameBytes)

	for i = int64(cnf.ResumeCounter.CurName); i < nameSize; i += bytesToRead {
		status := SyncSend(self, string(nameBytes[i:i+bytesToRead]), "info", true)
		if status != OK {
			return
		}
	}

	//------------------Передача текста из файла---------------------------------
	for i = int64(cnf.ResumeCounter.CurFile); i < fileSize; i += bytesToRead {
		sliceOfBytes := ReadFilePart(file, i, int(bytesToRead))
		status := SyncSend(self, string(sliceOfBytes), "info", true)
		if status != OK {
			return
		}
	}
	//TODO Отправка флага конца передачи
	//time.Sleep(time.Second)
	fmt.Println(time.Since(start))
	mu.Unlock()
	fmt.Println("\t- Mutex transmitResumeSlave")
}

func transmitDataSlave(self *conn, mu *sync.Mutex, DataFileNameWithPath string) {
	mu.Lock()
	fmt.Println("\t+ Mutex transmitDataSlave")
	start := time.Now()
	var fileSize, nameSize, i, bytesToRead int64
	var nameCadrSize int16
	var fileCadrSize int32
	file, err := os.Open(DataFileNameWithPath)
	CheckError(err)
	fmt.Println([]byte(DataFileNameWithPath))
	fmt.Println(DataFileNameWithPath)
	arr := strings.Split(DataFileNameWithPath, "/")
	fmt.Println(arr)
	DataFileName := arr[len(arr)-1]
	fmt.Println("FILENAME  ", DataFileName)
	stat, err := file.Stat()
	CheckError(err)
	fileSize = stat.Size()
	nameSize = int64(len(DataFileName))
	bytesToRead = 6
	fileCadrSize = int32(fileSize / bytesToRead)
	if int(fileSize)%int(bytesToRead) != 0 {
		fileCadrSize++
	}
	nameCadrSize = int16(nameSize / bytesToRead)
	if int(nameSize)%int(bytesToRead) != 0 {
		nameCadrSize++
	}
	fmt.Println("fileSize = " + strconv.Itoa(int(fileSize)) + ", fileCadrSize = " + strconv.Itoa(int(fileCadrSize)))
	fmt.Println("nameSize = " + strconv.Itoa(int(nameSize)) + ", nameCadrSize = " + strconv.Itoa(int(nameCadrSize)))
	//----------------------Инициализирующее сообщение---------------------------
	initMsg := GetInitMsg(fileCadrSize, nameCadrSize)
	status := SyncSend(self, string(initMsg), "transansinit", true)
	if status != OK {
		return
	}
	//----------------------Передача названия------------------------------------
	nameBytes := []byte(DataFileName)
	for len(nameBytes)%int(bytesToRead) != 0 { //TODO переписать, неэффективно
		nameBytes = append(nameBytes, 0)
	}
	//fmt.Printf("nameBytes: %b\n", nameBytes)

	for i = 0; i < nameSize; i += bytesToRead {
		status := SyncSend(self, string(nameBytes[i:i+bytesToRead]), "info", true)
		if status != OK {
			return
		}
	}

	//------------------Передача текста из файла---------------------------------
	for i = 0; i < fileSize; i += bytesToRead {
		sliceOfBytes := ReadFilePart(file, i, int(bytesToRead))
		status := SyncSend(self, string(sliceOfBytes), "info", true)
		if status != OK {
			return
		}

	}
	//TODO Отправка флага конца передачи
	//time.Sleep(time.Second)
	fmt.Println(time.Since(start))

	mu.Unlock()
	fmt.Println("\t- Mutex transmitDataSlave")
}

/*------------------------------------------------------------------------------*/

func CLIParser(stream chan string) {
	var res string
	for {
		//fmt.Print("command: ")
		fmt.Scanln(&res)
		if res == "OUT" {
			break
		}
		stream <- res
		fmt.Println("OK!\t command wrote ")
	}
}

type CNF struct {
	Name          string
	Baud          int
	FileName      string
	FileDir       string
	ResumeCounter counter
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

	arr := strings.Split(data.FileName, "/")
	DataFileName := arr[len(arr)-1]
	fmt.Println("FILENAME  ", DataFileName)
	buf, err := json.Marshal(*data)
	if err != nil {
		panic(err)
	}
	_, err = file.Write(buf)
	if err != nil {
		panic(err)
	}
}
