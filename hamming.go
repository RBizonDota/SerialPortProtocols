package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"
	"strconv"
	"time"
)

var DataFileName = "BigTextDoNotTransmit.txt"
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
		firstByte = append(firstByte, 240)
	case "noinfo":
		firstByte = append(firstByte, 42) // еще кейсы
	case "init":
		firstByte = append(firstByte, 122)
	case "end":
		firstByte = append(firstByte, 144)
	default:
		firstByte = append(firstByte, 42)
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

func main() {
	//var tmpArr []byte

	mychan := make(chan int64, 10)
	//-------------проверка получения------------------------
	go Getter(mychan)
	//-------------------------------------------------------

	//TODO Отправка флага начала передачи, длины названия в кадрах, длины тела в кадрах
	//TODO (кадр флаг 1-м байтом, два последних - длина названия и длина тела)
	//--------------Отправка файла----------------
	start := time.Now()
	var fileSize, nameSize, i, bytesToRead int64
	var nameCadrSize int16
	var fileCadrSize int32
	file, err := os.Open(DataFileName)
	CheckError(err)

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
	dataInBits := ToBits(initMsg)
	codedInitMsg := Code(dataInBits, bits.Len(uint(dataInBits)))
	mychan <- codedInitMsg

	//----------------------Передача названия------------------------------------
	nameBytes := []byte(DataFileName)
	for len(nameBytes)%int(bytesToRead) != 0 { //TODO переписать, неэффективно
		nameBytes = append(nameBytes, 0)
	}
	//fmt.Printf("nameBytes: %b\n", nameBytes)

	for i = 0; i < nameSize; i += bytesToRead {
		nameSlice := AddFrameType(nameBytes[i:i+bytesToRead], "info")
		//fmt.Printf("nameSlice: %b\n", nameSlice)
		dataInBits := ToBits(nameSlice)
		codedName := Code(dataInBits, bits.Len(uint(dataInBits)))
		mychan <- codedName
	}

	//------------------Передача текста из файла---------------------------------
	for i = 0; i < fileSize; i += bytesToRead {
		sliceOfBytes := ReadFilePart(file, i, int(bytesToRead))
		sliceOfBytes = AddFrameType(sliceOfBytes, "info")
		dataInBits := ToBits(sliceOfBytes)
		//fmt.Printf("dataBits: %064b\n", dataInBits)
		//fmt.Printf("dataBitsLen: %d\n", bits.Len(uint(dataInBits)))
		codedMsg := Code(dataInBits, bits.Len(uint(dataInBits)))
		//fmt.Printf("codedMsg: %064b\n\n", codedMsg)
		mychan <- codedMsg
	}
	//TODO Отправка флага конца передачи
	close(mychan)
	time.Sleep(time.Second)
	fmt.Println(time.Since(start))

	//Получение initMsg - в Getter
	//----------------Получение названия---------------------
	/*var receivedName []byte
	var fileName string
	for i = 0; i < nameSize; i += bytesToRead {
		fileNamePart := <-mychan
		decoded, _, _ := Decode(fileNamePart, bits.Len(uint(fileNamePart)))
		//проверка frameType и valid?
		//fmt.Printf("name: decoded:%08b, frameType:%d, valid:%t\n", decoded, frameType, valid)
		receivedName = append(receivedName, decoded...)
	}
	n := bytes.Index(receivedName, []byte{0})
	fileName = string(receivedName[:n]) //без конечных нулей
	fmt.Printf("Received fileName: %s\n", fileName)

	//---------------Получение файла------------------------
	var fileTextBytes []byte
	for i = 0; i < fileSize; i += bytesToRead {
		receivedStr := <-mychan
		decoded, _, _ := Decode(receivedStr, bits.Len(uint(receivedStr)))
		//проверка frameType и valid?
		fileTextBytes = append(fileTextBytes, decoded...)
	}
	fmt.Printf("Received fileText: %s\n", string(fileTextBytes))
	//------------------Запись в файл-------------------------
	m := bytes.Index(fileTextBytes, []byte{0}) //??????
	DataToFile(fileName, fileTextBytes[:m])    //???????????*/
}

func Getter(mychan chan int64) {
	//tmpArr := make([]byte, 0, 1)
	//TODO Обработка не закрытия канала, а флага начала и конца передачи
	msg, val := <-mychan
	if !val {
		fmt.Println("ERROR!!! chanal closed")
	}
	//TODO Анализ типа кадра
	decoded, _, _ := Decode(msg, bits.Len(uint(56)))
	//fmt.Printf("Init decoded:%08b, frameType:%d, valid:%t\n", decoded, frameType, valid)
	fileSizeSlice := decoded[0:4]
	nameSizeSlice := decoded[4:6]
	fileSize := binary.LittleEndian.Uint32(fileSizeSlice)
	nameSize := binary.LittleEndian.Uint16(nameSizeSlice)
	fmt.Printf("fileSize: %d\n", fileSize)
	fmt.Printf("nameSize: %d\n", nameSize)

	var receivedName []byte
	var fileName string
	for i := 0; i < int(nameSize); i++ {
		fileNamePart := <-mychan
		decoded, _, _ := Decode(fileNamePart, bits.Len(uint(fileNamePart)))
		//проверка frameType и valid?
		receivedName = append(receivedName, decoded...)
	}
	fmt.Println(string(receivedName))
	n := bytes.Index(receivedName, []byte{0})
	if n != -1 {
		fileName = string(receivedName[:n]) //без конечных нулей
	} else {
		fileName = string(receivedName)
	}
	fmt.Printf("Received fileName: %s\n", fileName)

	//---------------Получение файла------------------------
	var fileTextBytes []byte
	for i := 0; i < int(fileSize); i++ {
		receivedStr := <-mychan
		decoded, _, _ := Decode(receivedStr, bits.Len(uint(receivedStr)))
		//проверка frameType и valid?
		fileTextBytes = append(fileTextBytes, decoded...)
	}
	//fmt.Printf("Received fileText: %s\n", string(fileTextBytes))
	//------------------Запись в файл-------------------------
	//m := bytes.Index(fileTextBytes, []byte{0})
	//fmt.Println(string(fileTextBytes[:m]))
	//DataToFile(fileName, fileTextBytes[:m]) //без конечных нулей
	DataToFile(fileName, fileTextBytes)
	//fmt.Printf("decoded:%08b, valid:%t\n\n", decoded, valid)
	//tmpArr = append(tmpArr, decoded...)
	//fmt.Println(tmpArr)
	//fmt.Printf("Text: %s\n\n", string(tmpArr))
	//DataToFile("Test.txt", tmpArr)
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
	resinitMsg = AddFrameType(resinitMsg, "init")
	return resinitMsg
}
