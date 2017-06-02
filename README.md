# gorune
A Go library for reading the RuneScape filesystem format.

## Installation
In your terminal, execute `go get github.com/Velocity-/gorune`. Import in your files as usual: `import "github.com/Velocity-/gorune"`.

## Usage
### Loading a file system
```go
fs, err := gorune.Load("/path/to/directory", true) // Set to false to disable index scanning
if err != nil {
    panic(err)
}
```
You can also load an index manually, which adds the index to the filesystem and returns it:
```go
index, err := fs.LoadIndex(3, "/path/to/directory/main_file_cache.idx3")
...
```

### Reading data
You can read raw (possibly compressed) data from the filesystem using ReadRaw:
```go
entry := fs.Indices[3].Entries[12]
data, err := fs.ReadRaw(entry) // Will load raw data from index 3, entry 12.
...
```

To decompress said data, you can simply call `Decompress([]byte)`:
```go
decompressed, type, err := gorune.Decompress(compressed)
// type is the compression type used: 
// NoCompression, Bzip2Compression, GzipCompression.
```

To facilitate this process, you can also call `ReadDecompressed` to read raw data, and then decompress it:
```go
decompressed, err := fs.ReadDecompressed(fs.Indices[3].Entries[12])
```

### Reference table
The filesystem has a reference table used for describing how the entries are laid out and its attributes (such as CRC32, hashed name, files...). To load a descriptor:
```go
data, _ := fs.ReadDecompressed(fs.Indices[255].Entries[2])
referenceTable, err := gorune.DecodeReferenceTable(data)
fmt.Printf("Table data: %+v \n", referenceTable)
```