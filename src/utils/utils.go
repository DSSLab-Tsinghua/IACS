// 类型转换

package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"log"
	"strconv"
	"sync"
	"time"
)

type IntValue struct {
	m int
	sync.RWMutex
}

func (s *IntValue) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = 0
}

func (s *IntValue) Get() int {
	s.Lock()
	defer s.Unlock()
	return s.m
}

func (s *IntValue) Increment() int {
	s.Lock()
	defer s.Unlock()
	s.m = s.m + 1
	return s.m
}

func (s *IntValue) Set(n int) {
	s.Lock()
	defer s.Unlock()
	s.m = n
}

type BoolValue struct {
	m bool
	sync.RWMutex
}

func (s *BoolValue) Init() {
	s.Lock()
	defer s.Unlock()
	s.m = false
}

func (s *BoolValue) Get() bool {
	s.Lock()
	defer s.Unlock()
	return s.m
}

func (s *BoolValue) Set(n bool) {
	s.Lock()
	defer s.Unlock()
	s.m = n
}

type Vec struct {
	v []int
	sync.Mutex
}

func (s *Vec) Init(size int) {
	s.Lock()
	defer s.Unlock()
	var tmp []int
	s.v = tmp
	for i := 0; i < size; i++ {
		s.v = append(s.v, 0)
	}
}

func (s *Vec) Set(idx int, val int) {
	s.Lock()
	defer s.Unlock()
	s.v[idx] = val
}

func (s *Vec) Get() []int {
	s.Lock()
	defer s.Unlock()
	return s.v
}

func (s *Vec) GetIdx(idx int) int {
	s.Lock()
	defer s.Unlock()
	return s.v[idx]
}

func BytesToInt(bys []byte) int {
	bytebuff := bytes.NewBuffer(bys)
	var data int64
	binary.Read(bytebuff, binary.BigEndian, &data)
	return int(data)
}

func Int64ToString(input int64) string {
	return strconv.FormatInt(input, 10)
}

func StringToInt(input string) (int, error) {
	return strconv.Atoi(input)
}

func IntToString(input int) string {
	return strconv.Itoa(input)
}

func StringToInt64(input string) (int64, error) {
	return strconv.ParseInt(input, 10, 64)
}

func StringToBytes(input string) []byte {
	return []byte(input)
}

func MakeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func BytesToString(input []byte) string {
	return string(input[:])
}

func IntToInt64(input int) int64 {
	return int64(input)
}

func IntToBytes(n int) []byte {
	data := int64(n)
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, data)
	return bytebuf.Bytes()
}

func Int64ToInt(input int64) (int, error) {
	tmp := strconv.FormatInt(input, 10)
	output, err := strconv.Atoi(tmp)
	return output, err
}

func SerializeBytes(input [][]byte) []byte {
	if len(input) == 0 {
		return nil
	}
	var output []byte
	for i := 0; i < len(input); i++ {
		output = append(output, input[i]...)
	}
	return output
}

func IntsToBytes(ints []int) []byte {
	buf := new(bytes.Buffer)
	for _, i := range ints {
		if err := binary.Write(buf, binary.LittleEndian, int32(i)); err != nil {
			return nil
		}
	}
	return buf.Bytes()
}

func BytesToInts(data []byte) []int {
	ints := make([]int, len(data)/4)
	buf := bytes.NewReader(data)
	for i := range ints {
		var num int32
		if err := binary.Read(buf, binary.LittleEndian, &num); err != nil {
			return nil
		}
		ints[i] = int(num)
	}
	return ints
}

type Prejustify struct {
	Pre     int
	Justify map[int]int
}

func (s *Prejustify) Getpre() int {
	return s.Pre
}

func (s *Prejustify) Getjustify() map[int]int {
	return s.Justify
}

func (s *Prejustify) SetPrejustify(p int, j map[int]int) {
	s.Pre = p
	s.Justify = j
}

func (s *Prejustify) PrejustifyToBytes() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(s)
	if err != nil {
		log.Fatalf("failed to encode prejustify: %v", err)
	}
	return buf.Bytes()
}

func BytesToPrejustify(data []byte) Prejustify {
	var result Prejustify
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(&result)
	if err != nil {
		log.Fatalf("failed to decode prejustify: %v", err)
	}
	return result
}
