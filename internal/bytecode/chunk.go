package bytecode

import (
	"fmt"

	"icoo_lang/internal/runtime"
)

type Chunk struct {
	Code      []byte
	Constants []runtime.Value
	Lines     []int
}

func NewChunk() *Chunk {
	return &Chunk{
		Code:      make([]byte, 0, 128),
		Constants: make([]runtime.Value, 0, 32),
		Lines:     make([]int, 0, 128),
	}
}

func (c *Chunk) Write(op byte, line int) {
	c.Code = append(c.Code, op)
	c.Lines = append(c.Lines, line)
}

func (c *Chunk) WriteShort(v uint16, line int) {
	c.Code = append(c.Code, byte(v>>8), byte(v))
	c.Lines = append(c.Lines, line, line)
}

func (c *Chunk) AddConstant(v runtime.Value) uint16 {
	c.Constants = append(c.Constants, v)
	return uint16(len(c.Constants) - 1)
}

func (c *Chunk) GetConstant(index uint16) (runtime.Value, error) {
	if int(index) >= len(c.Constants) {
		return nil, fmt.Errorf("constant index out of range: %d", index)
	}
	return c.Constants[index], nil
}
