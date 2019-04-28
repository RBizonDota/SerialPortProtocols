package main

import (
	"fmt"
	"sync"
)

func manager(command string) {
	//меняет состояние работы соединения
	//боды, размер пакета, синхробайты
	fmt.Println("OK!\t command received " + command)
}

func goMaster() {
	//cnf := getCnf()
	cnfname := "cnf_master.json"
	cnf := getCnf(cnfname)
	fmt.Println(cnf)
	self := NewConn()
	fmt.Println("OK!\t   MASTER init")
	mu := &sync.Mutex{}
	go manageHandler(&self, mu, cnf, cnfname)
	CLIParser(self.ManageStream)
	return
}
func goSlave() {
	cnfname := "cnf_slave.json"
	cnf := getCnf(cnfname)
	fmt.Println(cnf)
	self := NewConn()
	fmt.Println("OK!\t   SLAVE init")
	mu := &sync.Mutex{}
	go manageHandler(&self, mu, cnf, cnfname)
	CLIParser(self.ManageStream)

}

/*func simpleSend(self *conn) {
	go func() {
		for {
			SyncSend(self, "Q", true)
			time.Sleep(time.Second)
		}
	}()
}*/
