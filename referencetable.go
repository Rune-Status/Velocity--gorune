package gorune

import (
	"fmt"
)

// ReferenceTable represents a table of data holding specifics about
// the filesystem index, such as the revision of entries, names and checksums.
type ReferenceTable struct {
	version    int8
	revision   int32
	entryCount int32
	ids        []int32
	size       int32
	names      []int32
	crc32      []int32
	whirlpool  [][]byte
	versions   []int32
}

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

	// If bit 8 is set (negative byte), the low 31 bits
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

// DecodeReferenceTable decodes (decompressed) data into a ReferenceTable struct.
func DecodeReferenceTable(data []byte) (*ReferenceTable, error) {
	buffer := ByteBuffer{data: data}
	table := ReferenceTable{}

	table.version = buffer.int8()
	if table.version >= 5 && table.version <= 7 {
		if table.version >= 6 {
			table.revision = buffer.int32()
		}

		flags := buffer.int8()
		hasNames := (flags & 1) != 0
		hasWhirlpool := (flags & 2) != 0
		unknown1 := (flags & 4) != 0
		unknown2 := (flags & 8) != 0

		if table.version >= 7 {
			table.entryCount = buffer.varint()
		} else {
			table.entryCount = int32(buffer.int16()) & 0xFFFF
		}
		table.ids = make([]int32, table.entryCount)

		id := int32(0)
		max := int32(-1)

		for i := int32(0); i < table.entryCount; i++ {
			// Type of data depends on the table version - only 7+ supports >65535
			if table.version >= 7 {
				id += buffer.varint()
			} else {
				id += int32(buffer.int16()) & 0xFFFF
			}

			table.ids[i] = id

			// Keep track of the highest id
			if id > max {
				max = id
			}
		}

		table.size = max + 1

		// Load all names, if present
		if hasNames {
			table.names = make([]int32, table.size)

			// Set all names to -1 (meaning no name)
			for i := int32(0); i < table.entryCount; i++ {
				table.names[table.ids[i]] = buffer.int32()
			}
		}

		// Load CRC values
		table.crc32 = make([]int32, table.size)
		for i := int32(0); i < table.entryCount; i++ {
			table.crc32[table.ids[i]] = buffer.int32()
		}

		// Unidentified
		if unknown2 {
			for i := int32(0); i < table.entryCount; i++ {
				buffer.int32()
			}
		}

		// Read whirlpool values
		if hasWhirlpool {
			table.whirlpool = make([][]byte, table.entryCount)
			for i := int32(0); i < table.entryCount; i++ {
				table.whirlpool[table.ids[i]] = buffer.bytes(64)
			}
		}

		// Unidentified
		if unknown1 {
			for i := int32(0); i < table.entryCount; i++ {
				buffer.int32()
				buffer.int32()
			}
		}

		// Load entry versions
		table.versions = make([]int32, table.size)
		for i := int32(0); i < table.entryCount; i++ {
			table.versions[table.ids[i]] = buffer.int32()
		}

		return &table, nil
	} else {
		return nil, fmt.Errorf("invalid reference table version: %d", table.version)
	}

	return nil, nil
}
