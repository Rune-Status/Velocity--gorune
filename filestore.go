package gorune

import (
	"encoding/binary"
	"fmt"
	"os"
)

const SectorLen = 520

type FileSystem struct {
	dataFile *os.File
	Indices  [256]*Index
}

// Index is a struct holding the entries loaded from the index file.
type Index struct {
	Entries []IndexEntry
}

// IndexEntry represents an entry in the index descriptor, specifying
// an entry's size and absolute offset in the main data file.
type IndexEntry struct {
	Id     uint32
	Size   uint32
	Offset uint64
}

// Load loads a filesystem in a given folder, optionally also loading all indices
// if findIndices is set. It is assumed that the main data file is called 'main_file_cache.dat2'.
func Load(folder string, findIndices bool) (*FileSystem, error) {
	dataFile, err := os.Open(folder + string(os.PathSeparator) + "main_file_cache.dat2")
	if err != nil {
		return nil, err
	}

	fs := &FileSystem{
		dataFile: dataFile,
	}

	if findIndices {
		fs.FindIndices(folder)
	}

	return fs, nil
}

// FindIndices tries to find as many indices as possible in the given folder.
// This will load the indices and parse them into memory.
func (fs *FileSystem) FindIndices(folder string) {
	for i := 0; i < 256; i++ {
		fs.LoadIndex(i, fmt.Sprintf("%s%cmain_file_cache.idx%d", folder, os.PathSeparator, i))
	}
}

// LoadIndex loads an index from disk and reads the entries into memory for faster
// lookups without needing to resort to disk operations. It is assumed that the
// index has size/6 entries, and that each entry is exactly 6 bytes.
func (fs *FileSystem) LoadIndex(id int, file string) (*Index, error) {
	indexFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	stat, err := indexFile.Stat()
	if err != nil {
		return nil, err
	}

	numEntries := stat.Size() / 6
	temp := make([]byte, numEntries*6)
	entries := make([]IndexEntry, numEntries)
	indexFile.Read(temp)

	// Put all the entry data into entries
	for i := int64(0); i < numEntries; i++ {
		b := temp[i*6:]
		size := uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
		offset := uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])

		entries[i] = IndexEntry{
			Id:     uint32(i),
			Size:   size,
			Offset: offset * SectorLen,
		}
	}

	index := &Index{
		Entries: entries,
	}
	fs.Indices[id] = index

	return index, nil
}

// ReadRaw reads raw data from the file system, given that the
// entry can be found. If the entry is not found, this method
// returns an error.
//
// Data read using this method can be decompressed by calling Decompress
// and both operations can also be done by calling ReadDecompressed.
func (fs *FileSystem) ReadRaw(entry IndexEntry) ([]byte, error) {
	if entry.Size < 1 {
		return nil, fmt.Errorf("entry %d has no data", entry.Id)
	}

	scratchBuffer := make([]byte, 8)
	dataBuffer := make([]byte, entry.Size)
	dataIndex := 0

	position := int64(entry.Offset)
	currentChunk := -1
	headerSize := 8
	if entry.Id > 0xFFFF {
		headerSize = 10
	}

	for remaining := int(entry.Size); remaining > 0; {
		fs.dataFile.ReadAt(scratchBuffer, position)

		var folderID uint32
		b := scratchBuffer

		if entry.Id > 0xFFFF {
			folderID = binary.BigEndian.Uint32(b)
			b = scratchBuffer[2:] // Offset by 2 to make the code below work fine.
		} else {
			folderID = uint32(binary.BigEndian.Uint16(b))
		}

		chunkID := binary.BigEndian.Uint16(b[2:])
		nextSector := uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])

		if folderID != entry.Id || int(chunkID) != currentChunk+1 {
			return nil, fmt.Errorf("malformed index entry %d: sequence does not complete [%d %d] v. [%d %d] %d",
				entry.Id, folderID, entry.Id, chunkID, currentChunk+1, position)
		}

		sizeToRead := SectorLen - headerSize
		if sizeToRead > remaining {
			sizeToRead = remaining
		}

		fs.dataFile.ReadAt(dataBuffer[dataIndex:dataIndex+sizeToRead], position+int64(headerSize))
		position = int64(nextSector * SectorLen)

		currentChunk++
		remaining -= sizeToRead
		dataIndex += sizeToRead
	}

	return dataBuffer, nil
}

// ReadDecompressed reads an entry's raw data, and then (if successful) proceeds
// to decompress the contents using Decompress(). If the decompression fails,
// no data is returned at all and an error from the decompression is returned.
func (fs *FileSystem) ReadDecompressed(entry IndexEntry) ([]byte, error) {
	raw, err := fs.ReadRaw(entry)
	if err != nil {
		return nil, err
	}

	decompressed, _, err := Decompress(raw)
	return decompressed, err
}
