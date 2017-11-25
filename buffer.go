package gorune

type ByteBuffer struct {
	data   []byte
	offset int
}

func (b *ByteBuffer) int8() int8 {
	b.offset++
	return int8(b.data[b.offset-1])
}

func (b *ByteBuffer) int16() int16 {
	b.offset += 2
	return int16(b.data[b.offset-2])<<8 | int16(b.data[b.offset-1])
}

func (b *ByteBuffer) int32() int32 {
	b.offset += 4
	return int32(b.data[b.offset-4])<<24 | int32(b.data[b.offset-3])<<16 | int32(b.data[b.offset-2])<<8 | int32(b.data[b.offset-1])
}

func (b *ByteBuffer) varint() int32 {
	first := b.data[b.offset]

	// If bit 8 is set (sign bit), the low 31 bits
	// of the current 4 bytes represent an int32, otherwise
	// they represent an int16.
	if first < 0 {
		return b.int32() & 0x7FFFFFFF
	} else {
		return int32(b.int16()) & 0xFFFF
	}
}

func (b *ByteBuffer) bytes(num int) []byte {
	out := make([]byte, num)
	copy(out, b.data[b.offset:b.offset+num])
	return out
}