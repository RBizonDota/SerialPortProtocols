package main

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"
)

var DataFileName = "BigTextDoNotTransmit.txt"
var DataFolder = "data/"

var MasksForZeros = []int64{0x7FFFFFFFF0000000, 0xFFFE000, 0x1FC0, 0x38, 0x7}

var MainMasks = []int64{0x5555555555555555, 0x6666666666666666, 0x7878787878787878,
	0x7F807F807F807F80, 0x7FFF80007FFF8000, 0x7FFFFFFF80000000}

var DeleteBitsMasks = []int64{0x7FFFFFFF00000000, 0x7FFF0000, 0x7F00, 0x70, 0x4}

const (
	syncCadr         byte = 228
	accCadr          byte = 10
	connInit         byte = 33
	connEnd          byte = 22
	infoCadr         byte = 240
	transInitCadr    byte = 150
	transAnsInitCadr byte = 122
	transEndCadr     byte = 144
	defaultCadr      byte = 42
	transResumeCadr  byte = 55
)

func CheckError(err error) { //remake
	if err != nil {
		fmt.Print(err)
	}
}

func FileNameToBits() {

}

func ReadFilePart(f *os.File, placeFrom int64, numOfBytes int) []byte {
	_, err := f.Seek(placeFrom, 0)
	CheckError(err)
	sliceBytes := make([]byte, numOfBytes)
	_, err = f.Read(sliceBytes)
	CheckError(err)
	//fmt.Printf("sliceBytes: %s\n", string(sliceBytes))
	return sliceBytes
}

func AddFrameType(bytesArr []byte, frameType string) []byte {
	var firstByte []byte
	switch frameType {
	case "info":
		firstByte = append(firstByte, infoCadr)
	case "noinfo":
		firstByte = append(firstByte, defaultCadr)

	case "transansinit":
		firstByte = append(firstByte, transAnsInitCadr)
	case "transinit":
		firstByte = append(firstByte, transInitCadr)
	case "transend":
		firstByte = append(firstByte, transEndCadr)
	case "init":
		firstByte = append(firstByte, connInit)
	case "end":
		firstByte = append(firstByte, connEnd)
	case "sync":
		firstByte = append(firstByte, syncCadr)
	case "acc":
		firstByte = append(firstByte, accCadr)
	default:
		firstByte = append(firstByte, defaultCadr)
	}

	bytesArr = append(firstByte, bytesArr...)
	return bytesArr
}

func ToBits(sliceBytes []byte) int64 {
	var bits int64
	//fmt.Println(sliceBytes)
	//TODO сделать отслеживание размера кадров (если пришло меньше 6, то дополнить 0-ми байтами)
	for _, b := range sliceBytes {
		bits <<= 8
		bits += int64(b)
	}
	return bits
}

func InsertZeros(msg int64) int64 {
	var msgWithZeros int64
	msg <<= 2
	for _, mask := range MasksForZeros {
		msgWithZeros <<= 1
		msgWithZeros += msg & mask //или  |=
	}
	return msgWithZeros
}

func Code(msg int64, msgLen int) int64 {
	var position uint = 1
	msg = InsertZeros(msg)
	//fmt.Printf("withZero: %064b\n", msg)

	for i := 0; i < len(MainMasks); i++ {
		controlBit := CountControlBit(msg, MainMasks[i], position)
		controlBit <<= position - 1
		msg |= controlBit //вставка контольного бита
		position *= 2
	}
	return msg
}

func CountControlBit(bitsMsg int64, mask int64, position uint) int64 {
	var controlBit int64
	resultMask := bitsMsg & mask

	if bits.OnesCount(uint(resultMask))%2 != 0 {
		controlBit = 1
	}
	return controlBit
}

func Decode(msg int64, msgLen int) ([]byte, byte, bool) {
	var valid bool = true
	var syndr int64
	//var errorVector int64 = 1
	var position uint = 1

	for i := 0; i < len(MainMasks); i++ {
		syndrBit := CountControlBit(msg, MainMasks[i], position)
		syndrBit <<= uint(i)
		syndr |= syndrBit
		position *= 2
	}
	//fmt.Println("syndr: ", syndr)

	if syndr != 0 {
		//исправить 1ую ошибку
		// errorVector <<= uint(syndr) - 1
		// msg ^= errorVector
		valid = false
	}

	msg = DeleteControlBits(msg)
	sliceBytes, frameType := ToBytes(msg)

	return sliceBytes, frameType, valid
}

func DeleteControlBits(msg int64) int64 {
	var msgBits int64

	for _, mask := range DeleteBitsMasks {
		msgBits >>= 1
		msgBits += msg & mask
	}

	msgBits >>= 2 //1 2 биты
	return msgBits
}

func GetFrameType(msg int64) (int64, byte) {
	var mask int64 = 0x7F80000000000000
	frameType := byte((msg & mask) >> 55)
	msg <<= 8
	return msg, frameType
}

func ToBytes(msg int64) ([]byte, byte) {
	var curBit int64
	var curByte byte
	var mask int64 = 0x7F80000000000000
	var sliceBytes []byte

	msg <<= 7 //старший байт не используется
	msg, frameType := GetFrameType(msg)

	for len(sliceBytes) != 6 {
		curBit = msg & mask
		curBit >>= 55
		curByte = byte(curBit)
		sliceBytes = append(sliceBytes, curByte)
		msg <<= 8
	}

	//fmt.Println("sliceBytes in ToBytes: ", sliceBytes)
	return sliceBytes, frameType
}

//DataToFile true - все хорошо,false - возникла ошибка; ТРЕБУЕТ СУЩЕСТВОВАНИЯ ДИРЕКТОРИИ!!!
func DataToFile(filename string, body []byte, dirname string) bool {
	if dirname == "" {
		fmt.Println("ERROR!!!   dirname not set")
		return false
	}
	file, err := os.Create(dirname + filename)
	if err != nil {
		fmt.Println("ERROR!!!   Unable to create file")
		fmt.Println(err.Error())
		return false
	}
	defer file.Close()
	_, err = file.Write(body)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(len(body))
	return true
}

func GetInitMsg(fileCadrSize int32, nameCadrSize int16) []byte {
	initMsgBytes1 := make([]byte, 4) //fileCadrSize [7 0]
	initMsgBytes2 := make([]byte, 2) //nameCadrSize [2 0]
	//WARNING МЕНЯЕТ БАЙТЫ МЕСТАМИ, СНАЧАЛА МЛАДШИЕ, ПОТОМ СТАРШИЕ
	binary.LittleEndian.PutUint32(initMsgBytes1, uint32(fileCadrSize))
	binary.LittleEndian.PutUint16(initMsgBytes2, uint16(nameCadrSize))
	resinitMsg := append(initMsgBytes1, initMsgBytes2...)
	return resinitMsg
}

func delEndZeros(data []byte) []byte {
	//fmt.Println(data)
	//fmt.Println(len(data))
	for i := len(data) - 1; i > -1; i-- {
		//fmt.Println("i = " + strconv.Itoa(i))
		if data[i] != 0 {
			return data[:i+1]
		}
	}
	return nil
}

func CreateFile(filename string, dirname string) bool {
	if dirname == "" {
		fmt.Println("ERROR!!!   dirname not set")
		return false
	}
	file, err := os.Create(dirname + filename)
	if err != nil {
		fmt.Println("ERROR!!!   Unable to create file")
		fmt.Println(err.Error())
		return false
	}
	fmt.Println(filename + " created")
	defer file.Close()
	return true
}

func AddDataToFile(filename string, body []byte, dirname string) bool {
	if dirname == "" {
		fmt.Println("ERROR!!!   dirname not set")
		return false
	}
	file, err := os.OpenFile(dirname+filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("ERROR!!!   Unable to open file")
		fmt.Println(err.Error())
		return false
	}
	defer file.Close()
	_, err = file.Write(body)
	if err != nil {
		fmt.Println(err.Error())
	}
	//fmt.Println(len(body))
	return true
}
