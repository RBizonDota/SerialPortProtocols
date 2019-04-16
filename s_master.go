package main

import (
	"fmt"
	"log"

	"github.com/mikepb/go-serial"
)

func main() {
	options := serial.RawOptions
	options.BitRate = 115200
	p, err := options.Open("COM1")
	if err != nil {
		log.Panic(err)
	}

	fmt.Println(p.DTR())
	fmt.Println(p.DSR())
	fmt.Println(p.RTS())
	fmt.Println(p.CTS())
	defer p.Close()

}
