package main

import (
	"fmt"
	"log"
	"net"
	"os"
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
	//CLIParser(self.ManageStream)
	f, err := os.OpenFile("log_master.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	listner, err := net.Listen("tcp", ":8888")
	fmt.Println("listener 8888 started")
	if err != nil {
		panic(err)
	}
	conn, err := listner.Accept()
	if err != nil {
		panic(err)
	}
	self.callBack = &conn
	testHandleConnection(conn, "MASTER", self.ManageStream, cnf)
}
func goSlave() {
	cnfname := "cnf_slave.json"
	cnf := getCnf(cnfname)
	fmt.Println(cnf)
	self := NewConn()
	fmt.Println("OK!\t   SLAVE init")
	mu := &sync.Mutex{}
	go manageHandler(&self, mu, cnf, cnfname)
	//CLIParser(self.ManageStream)
	f, err := os.OpenFile("log_slave.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)
	listner, err := net.Listen("tcp", ":8889")
	if err != nil {
		panic(err)
	}
	fmt.Println("listener 8889 started")
	conn, err := listner.Accept()
	if err != nil {
		panic(err)
	}
	self.callBack = &conn
	testHandleConnection(conn, "SLAVE ", self.ManageStream, cnf)
}

/*func simpleSend(self *conn) {
	go func() {
		for {
			SyncSend(self, "Q", true)
			time.Sleep(time.Second)
		}
	}()
}*/
