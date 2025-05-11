package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

type Walker struct {
	data    *ExtendBuffer
	posit   uint32
	littleE bool
	tagPath string
}

type ExtendBuffer struct {
	extendBy uint32
	reader   *bufio.Reader
	buff     []uint8
	length   uint32
}

func NewExtendBuffer(reader *bufio.Reader, extendBy uint32) *ExtendBuffer {
	eb := &ExtendBuffer{
		reader:   reader,
		buff:     make([]uint8, 0),
		length:   0,
		extendBy: extendBy,
	}
	eb.extend(0, 0)
	return eb
}

func (p *ExtendBuffer) extend(required uint32, pos uint32) {
	buffer := make([]byte, required+p.extendBy)
	lenRead, err := p.reader.Read(buffer)
	if err != nil {
		panic(fmt.Sprintf("Failed to extend buffer. Required %d. Current %d, Only able to read %d. Error: %s", required, pos, lenRead, err.Error()))
	}
	if required > uint32(lenRead) {
		panic(fmt.Sprintf("Failed to extend buffer. Required %d. Current %d, Only able to read %d.", required, pos, lenRead))
	}
	p.buff = append(p.buff, buffer[:lenRead]...)
	p.length = p.length + uint32(lenRead)
}

func NewWalker(reader *bufio.Reader, extendBy uint32) (*Walker, error) {
	buffer := NewExtendBuffer(reader, extendBy)
	return &Walker{
		data:    buffer,
		posit:   0,
		littleE: false,
		tagPath: "*",
	}, nil
}

func (p *Walker) Clone() *Walker {
	return &Walker{
		data:    p.data,
		posit:   p.posit,
		littleE: p.littleE,
		tagPath: p.tagPath,
	}
}

func (p *Walker) CloneWithPath(path string) *Walker {
	return &Walker{
		data:    p.data,
		posit:   p.posit,
		littleE: p.littleE,
		tagPath: path,
	}
}

func (p *Walker) SetLittleE(yes bool) {
	p.littleE = yes
}

func (p *Walker) Zstring(max int) string {
	var line bytes.Buffer
	count := 0
	for count < max {
		b := p.Advance(1)
		if b > 0 {
			line.WriteByte(byte(b))
		} else {
			return line.String()
		}
		count++
	}
	return line.String()
}

func (p *Walker) ZstringEquals(s string) bool {
	return (strings.EqualFold(p.Zstring(len(s)+2), s))
}

func (p *Walker) Bytes(n uint32) []byte {
	b := make([]byte, n)
	for i := 0; uint32(i) < n; i++ {
		b[i] = byte(p.Advance(1))
	}
	return b
}

func (p *Walker) Hex(b []byte, pre string) string {
	return pre + bytesToHex(b, 0)
}

func (p *Walker) BytesToUint(b []byte) uint64 {
	if p.littleE {
		return p.bytesToUintLE(b)
	}
	return p.bytesToUintBE(b)
}

func (p *Walker) BytesToInt(b []byte) int64 {
	if p.littleE {
		return int64(p.bytesToUintLE(b))
	}
	return int64(p.bytesToUintBE(b))
}

func (p *Walker) bytesToUintLE(b []byte) uint64 {
	switch len(b) {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.LittleEndian.Uint16(b))
	case 4:
		return uint64(binary.LittleEndian.Uint32(b))
	case 8:
		return uint64(binary.LittleEndian.Uint64(b))
	}
	return 0
}

func (p *Walker) bytesToUintBE(b []byte) uint64 {
	switch len(b) {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.BigEndian.Uint16(b))
	case 4:
		return uint64(binary.BigEndian.Uint32(b))
	case 8:
		return uint64(binary.BigEndian.Uint64(b))
	}
	return 0
}

func (p *Walker) Char() string {
	b := p.Advance(1)
	if b < 32 {
		return string(rune(1))
	}
	return string(byte(b & 0xFF))
}

func (p *Walker) Advance(n uint32) uint32 {
	p.ensurePos(p.posit)
	b := p.data.buff[p.posit]
	p.posit = p.posit + n
	return uint32(b) & 0xFF
}

func (p *Walker) Pos(n uint32) *Walker {
	p.ensurePos(n)
	p.posit = n
	return p
}

func (p *Walker) ensurePos(pos uint32) {
	if pos >= p.data.length {
		required := pos - (p.data.length - 1)
		p.data.extend(required, p.posit)
	}
}

func (q *Walker) LinePrint(start uint32, count int, lines int) string {
	clone := q.Clone()
	var line bytes.Buffer
	clone.Pos(start)
	for j := 0; j < lines; j++ {
		line.WriteString(pad(uint32(clone.posit), 4))
		line.WriteRune(':')
		line.WriteRune(' ')
		p := clone.posit
		for i := 0; i < count; i++ {
			line.WriteString(clone.Hex(clone.Bytes(1), ""))
			line.WriteRune(' ')
		}
		clone.Pos(p)
		for i := 0; i < (count / 2); i++ {
			line.WriteString(pad(uint32(clone.BytesToUint(clone.Bytes(2))), 6))
			line.WriteRune(' ')
		}
		line.WriteString("\n")
		clone.Pos(p)
		line.WriteString("    : ")
		for i := 0; i < count; i++ {
			line.WriteString(clone.Char())
			line.WriteRune(' ')
			line.WriteRune(' ')
		}
		clone.Pos(p)
		for i := 0; i < count/2; i++ {
			line.WriteString(clone.Hex(clone.Bytes(2), ""))
			line.WriteString("   ")
		}
		line.WriteString("\n")
	}
	return line.String()
}

func pad(i uint32, n int) string {
	s := fmt.Sprintf("%d", i)
	if len(s) >= n {
		return s
	}
	return "00000000000000000"[0:n-len(s)] + s
}

/*
Because I want to see hev values in upper case!
*/

var hexDigits = []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func bytesToHex(b []byte, delim rune) string {
	var line bytes.Buffer
	for i := 0; i < len(b); i++ {
		line.WriteString(byteToHex(b[i]))
		if delim > 32 && i < (len(b)-1) {
			line.WriteRune(delim)
		}
	}
	return line.String()
}

func byteToHex(b byte) string {
	var l bytes.Buffer
	l.WriteRune(hexDigits[b>>4])
	l.WriteRune(hexDigits[b&0x0f])
	return l.String()
}

func bytesToZString(b []byte) string {
	var line bytes.Buffer
	for _, c := range b {
		if c == 0 {
			break
		}
		line.WriteByte(c)
	}
	return line.String()
}
