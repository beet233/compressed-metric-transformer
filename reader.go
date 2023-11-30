package main

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type DataReader struct {
	data []byte
}

func NewDataReader(data []byte) *DataReader {
	return &DataReader{data: data}
}

func (r *DataReader) readString() (string, error) {
	if len(r.data) == 0 {
		return "", errors.New("no data available")
	}

	index := bytes.IndexByte(r.data, 0)
	if index == -1 {
		return "", errors.New("no null terminator found")
	}

	str := string(r.data[:index])
	r.data = r.data[index+1:]
	return str, nil
}

func (r *DataReader) readInt() (uint64, error) {
	if len(r.data) < 8 {
		return 0, errors.New("not enough data for int")
	}

	val := binary.LittleEndian.Uint64(r.data[:8])
	r.data = r.data[8:]
	return val, nil
}

// TODO
func (r *DataReader) readLeb128Int() (uint64, error) {
	return 0, nil
}

func (r *DataReader) readFloat() (float64, error) {
	if len(r.data) < 8 {
		return 0, errors.New("not enough data for float")
	}

	val := float64(binary.LittleEndian.Uint64(r.data[:8]))
	r.data = r.data[8:]
	return val, nil
}

func (r *DataReader) readByte() (byte, error) {
	if len(r.data) == 0 {
		return 0, errors.New("no data available")
	}

	val := r.data[0]
	r.data = r.data[1:]
	return val, nil
}

// func main() {
// 	// 示例数据
// 	data := []byte("Hello\x002345\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
//
// 	// 创建一个 DataReader 实例
// 	reader := NewDataReader(data)
//
// 	// 使用不同方法读取数据
// 	if str, err := reader.readString(); err == nil {
// 		fmt.Println("Read String:", str)
// 	}
//
// 	if num, err := reader.readInt(); err == nil {
// 		fmt.Println("Read Int:", num)
// 	}
//
// 	if floatNum, err := reader.readFloat(); err == nil {
// 		fmt.Println("Read Float:", floatNum)
// 	}
//
// 	if byteVal, err := reader.readByte(); err == nil {
// 		fmt.Println("Read Byte:", byteVal)
// 	}
// }
