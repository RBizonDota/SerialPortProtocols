package main

import (
	"fmt"
	"math/bits"
	"os"
	"strconv"
	"time"
)

//TODO не забыть задать нормальные флаги
var DataTransmitStart = byte(122)
var DataTransmitEnd = byte(144)

var DataFileName = "text.txt"
var DataFolder = "data/"

var MasksForZeros = []int64{0x7FFFFFFFF0000000, 0xFFFE000, 0x1FC0, 0x38, 0x7}

var MainMasks = []int64{0x5555555555555555, 0x6666666666666666, 0x7878787878787878,
	0x7F807F807F807F80, 0x7FFF80007FFF8000, 0x7FFFFFFF80000000}

var DeleteBitsMasks = []int64{0x7FFFFFFF00000000, 0x7FFF0000, 0x7F00, 0x70, 0x4}

func CheckError(err error) { //remake
	if err != nil {
		fmt.Print(err)
	}
}

func ReadFilePart(f *os.File, placeFrom int64, numOfBytes int) []byte {
	_, err := f.Seek(placeFrom, 0)
	CheckError(err)
	sliceBytes := make([]byte, numOfBytes)
	_, err = f.Read(sliceBytes)
	CheckError(err)
	fmt.Printf("sliceBytes: %s\n", string(sliceBytes))
	return sliceBytes
}

func AddFrameType(bytesArr []byte, frameType string) []byte {
	var firstByte []byte
	switch frameType {
	case "info":
		firstByte = append(firstByte, 240)
	case "noinfo":
		firstByte = append(firstByte, 42) // еще кейсы
	default:
		firstByte = append(firstByte, 42)
	}

	bytesArr = append(firstByte, bytesArr...)
	return bytesArr
}

func ToBits(sliceBytes []byte) int64 {
	var bits int64
	sliceBytes = AddFrameType(sliceBytes, "info")
	fmt.Println(sliceBytes)

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

func Decode(msg int64, msgLen int) ([]byte, bool) {
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
	fmt.Println("syndr: ", syndr)

	if syndr != 0 {
		//исправить 1ую ошибку
		// errorVector <<= uint(syndr) - 1
		// msg ^= errorVector
		valid = false
	}

	msg = DeleteControlBits(msg)
	sliceBytes := ToBytes(msg)

	return sliceBytes, valid
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

func ToBytes(msg int64) []byte {
	var curBit int64
	var curByte byte
	var frameType byte
	var sliceBytes []byte
	var mask int64 = 0x7F80000000000000
	msg <<= 7

	//получить тип кадра
	frameType = byte((msg & mask) >> 55)
	msg <<= 8
	fmt.Printf("frameType: %d\n", frameType)

	for len(sliceBytes) != 6 {
		curBit = msg & mask
		curBit >>= 55
		curByte = byte(curBit)
		sliceBytes = append(sliceBytes, curByte)
		msg <<= 8
	}

	fmt.Println("sliceBytes: ", sliceBytes)
	return sliceBytes
}

func main() {
	//var tmpArr []byte

	mychan := make(chan int64, 10)
	//-------------проверка получения------------------------
	go Getter(mychan)
	//-------------------------------------------------------

	//TODO Отправка флага начала передачи, длины названия в кадрах, длины тела в кадрах
	//TODO (кадр флаг 1-м байтом, два последних - длина названия и длина тела)
	//--------------Отправка файла----------------
	var fileSize, nameSize, i, bytesToRead int64
	var fileCadrSize, nameCadrSize byte
	file, err := os.Open(DataFileName)
	CheckError(err)

	stat, err := file.Stat()
	CheckError(err)
	fileSize = stat.Size()
	nameSize = int64(len(DataFileName))
	fmt.Printf("FileSize: %d\n", fileSize)
	bytesToRead = 6
	fileCadrSize = byte(fileSize/bytesToRead + 1)
	nameCadrSize = byte(nameSize/bytesToRead + 1)
	fmt.Println("fileSize = " + strconv.Itoa(int(fileSize)) + ", fileCadrSize = " + strconv.Itoa(int(fileCadrSize)))
	fmt.Println("nameSize = " + strconv.Itoa(int(nameSize)) + ", nameCadrSize = " + strconv.Itoa(int(nameCadrSize)))
	//TODO сбор кадра начала передачи данные в переменных fileCadrSize и nameCadrSize
	//TODO цикл передачи названия
	for i = 0; i < fileSize; i += bytesToRead {
		sliceOfBytes := ReadFilePart(file, i, int(bytesToRead))
		dataInBits := ToBits(sliceOfBytes)
		fmt.Printf("dataBits: %064b\n", dataInBits)
		fmt.Printf("dataBitsLen: %d\n", bits.Len(uint(dataInBits)))
		codedMsg := Code(dataInBits, bits.Len(uint(dataInBits)))
		fmt.Printf("codedMsg: %064b\n\n", codedMsg)
		mychan <- codedMsg
	}
	//TODO Отправка флага конца передачи
	close(mychan)
	time.Sleep(time.Second)

	//---------------Получение файла------------------------
	// var bytesArr []byte
	// for ... {
	// 	receivedStr, sndr := Decode(vector)
	// 	if sndr != 0 {
	// 		correctedStr := CorrectErr(receivedStr, sndr)
	// 	}
	// 	bytesArr = append(bytesArr, bitsToBytesArr(correctedStr))
	// }
	// text := string(bytesArr)
	// //создать и записать в файл
}

func Getter(mychan chan int64) {
	tmpArr := make([]byte, 0, 1)
	for {
		//TODO Обработка не закрытия канала, а флага начала и конца передачи
		msg, val := <-mychan
		if !val {
			break
		}
		decoded, valid := Decode(msg, bits.Len(uint(56)))
		fmt.Printf("decoded:%08b, valid:%t\n\n", decoded, valid)
		tmpArr = append(tmpArr, decoded...)

	}
	//TODO Сохранение файла в папку /data
	//TODO сделать /data - выносной переменной DataFolder
	fmt.Println(tmpArr)
	fmt.Printf("Text: %s\n\n", string(tmpArr))
	DataToFile("Test.txt", tmpArr)
}

//DataToFile true - все хорошо,false - возникла ошибка; ТРЕБУЕТ СУЩЕСТВОВАНИЯ ДИРЕКТОРИИ!!!
func DataToFile(filename string, body []byte) bool {
	file, err := os.Create(DataFolder + filename)
	if err != nil {
		fmt.Println("ERROR!!!   Unable to create file")
		fmt.Println(err.Error())
		return false
	}
	defer file.Close()
	file.Write(body)
	return true
}
