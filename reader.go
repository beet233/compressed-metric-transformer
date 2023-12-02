package main

import (
	"encoding/binary"
	"errors"
	"math"
)

type DataReader struct {
	data []byte
}

func NewDataReader(data []byte) *DataReader {
	return &DataReader{data: data}
}

func (r *DataReader) readString(length int) (string, error) {
	if len(r.data) < length {
		return "", errors.New("no data available")
	}

	str := string(r.data[:length])
	r.data = r.data[length:]
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

func (r *DataReader) readLeb128Int() (uint64, error) {
	var result uint64 = 0
	shift := 0
	for i := 0; i < 8; i++ {
		b, err := r.readByte()
		if err != nil {
			return 0, errors.New("not enough data for leb128 int")
		}
		result |= (uint64(b&0x7F) << shift)
		if (b & 0x80) == 0 {
			break
		}
		shift += 7
	}
	if shift == 56 {
		b, err := r.readByte()
		if err != nil {
			return 0, errors.New("not enough data for leb128 int")
		}
		// 最后一 byte 没有标记位
		result |= (uint64(b&0xFF) << shift)
	}
	return result, nil
}

func (r *DataReader) readFloat() (float64, error) {
	if len(r.data) < 8 {
		return 0, errors.New("not enough data for float")
	}

	val := math.Float64frombits(binary.LittleEndian.Uint64(r.data[:8]))
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
