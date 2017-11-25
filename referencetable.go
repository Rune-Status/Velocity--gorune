package gorune

import (
	"fmt"
)

// ReferenceTable represents a table of data holding specifics about
// the filesystem index, such as the revision of entries, names and checksums.
type ReferenceTable struct {
	Version    int8
	Revision   int32
	EntryCount int32
	ids        []int32
	size       int32
	names      []int32
	crc32      []int32
	whirlpool  [][]byte
	versions   []int32
}

// DecodeReferenceTable decodes (decompressed) data into a ReferenceTable struct.
func DecodeReferenceTable(data []byte) (*ReferenceTable, error) {
	buffer := ByteBuffer{data: data}
	table := ReferenceTable{}

	table.Version = buffer.int8()
	if table.Version >= 5 && table.Version <= 7 {
		if table.Version >= 6 {
			table.Revision = buffer.int32()
		}

		flags := buffer.int8()
		hasNames := (flags & 1) != 0
		hasWhirlpool := (flags & 2) != 0
		unknown1 := (flags & 4) != 0
		unknown2 := (flags & 8) != 0

		if table.Version >= 7 {
			table.EntryCount = buffer.varint()
		} else {
			table.EntryCount = int32(buffer.int16()) & 0xFFFF
		}
		table.ids = make([]int32, table.EntryCount)

		id := int32(0)
		max := int32(-1)

		for i := int32(0); i < table.EntryCount; i++ {
			// Type of data depends on the table version - only 7+ supports >65535
			if table.Version >= 7 {
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
			for i := int32(0); i < table.EntryCount; i++ {
				table.names[table.ids[i]] = buffer.int32()
			}
		}

		// Load CRC values
		table.crc32 = make([]int32, table.size)
		for i := int32(0); i < table.EntryCount; i++ {
			table.crc32[table.ids[i]] = buffer.int32()
		}

		// Unidentified
		if unknown2 {
			for i := int32(0); i < table.EntryCount; i++ {
				buffer.int32()
			}
		}

		// Read whirlpool values
		if hasWhirlpool {
			table.whirlpool = make([][]byte, table.size)
			for i := int32(0); i < table.EntryCount; i++ {
				table.whirlpool[table.ids[i]] = buffer.bytes(64)
			}
		}

		// Unidentified
		if unknown1 {
			for i := int32(0); i < table.EntryCount; i++ {
				buffer.int32()
				buffer.int32()
			}
		}

		// Load entry versions
		table.versions = make([]int32, table.size)
		for i := int32(0); i < table.EntryCount; i++ {
			table.versions[table.ids[i]] = buffer.int32()
		}

		return &table, nil
	} else {
		return nil, fmt.Errorf("invalid reference table version: %d", table.Version)
	}

	return nil, nil
}
