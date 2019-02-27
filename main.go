package main

import (
	"fmt"
	"sync"
	"time"
)

func manager(command string) {
	//меняет состояние работы соединения
	//боды, размер пакета, синхробайты
	fmt.Println("OK!\t command received " + command)
}

func goMaster() {
	self := NewConn()
	mu := &sync.Mutex{}
	go func() {
		for {
			command := <-self.ManageStream
			manageHandler(command, &self, mu)
		}
	}()
	fmt.Println("OK!\t MASTER init")
	//openPort(&self, MASTER)
	//self.ConnStatus = OK //должно получаться вследствие ConnectInit
	/*if self.PortStatus == OK {
		fmt.Println("OK!\t Port MASTER opened")
		go syncSignal(&self, mu)
	} else {
		fmt.Println("FAIL!\t UNABLE TO OPEN PORT MASTER!!!")
		return
	}*/
	return
}
func goSlave() {
	self := NewConn()
	fmt.Println("OK!\t SLAVE init")
	openPort(&self, SLAVE)
	//self.ConnStatus = OK
	if self.PortStatus == OK {
		fmt.Println("OK!\t Port SLAVE opened")
		go func() {
			//fmt.Println(self.ConnStatus)
			//		for self.ConnStatus == OK {
			//fmt.Println("   OK!\tinited")
			SyncRead(&self) //val
			//fmt.Println("\tOK!\tstatus" + strconv.Itoa(val))
			//		}

		}()
		//проверка кадра на тип
		//обработка служебных кадров
	} else {
		fmt.Println("FAIL!\t UNABLE TO OPEN PORT SLAVE!!!")
	}

}
func main() {
	go goSlave()
	go goMaster()
	defer fmt.Scanln()
}

func simpleSend(self *conn) {
	go func() {
		for {
			SyncSend(self, "Q")
			time.Sleep(time.Second)
		}
	}()
}
