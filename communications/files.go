package communications

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
)

var prefix = "_SharedFiles"

type Metadata struct {
	Filename string
	FileSize int64
	MetaFile []byte
	MetaHash [32]byte
}

// code taken from: https://www.socketloop.com/tutorials/golang-how-to-split-or-chunking-a-file-to-smaller-pieces
func (gossiper *Gossiper) GetMetadata(filename string) *Metadata {
	file, err := os.Open(prefix + "/" + gossiper.Name + "/" + filename)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	var fileSize int64 = fileInfo.Size()
	const fileChunk = 8 * (1 << 10) // 8 KB, change this to your requirement

	// calculate total number of parts the file will be chunked into
	totalPartsNum := uint64(math.Ceil(float64(fileSize) / float64(fileChunk)))
	fmt.Printf("Splitting to %d pieces.\n", totalPartsNum)
	metafile := make([]byte, 0)
	for i := uint64(0); i < totalPartsNum; i++ {
		// the last chunk will be smaller than 8 KB
		partSize := int(math.Min(fileChunk, float64(fileSize-int64(i*fileChunk))))
		partBuffer := make([]byte, partSize)

		file.Read(partBuffer)

		// write to disk
		//fileName := "somebigfile_" + strconv.FormatUint(i, 10)
		//_, err := os.Create(fileName)
		//
		//if err != nil {
		//	fmt.Println(err)
		//	os.Exit(1)
		//}

		// write/save buffer to disk
		//ioutil.WriteFile(prefix + "/" + gossiper.Name + "/" + filename, partBuffer, os.ModeAppend)
		//fmt.Println("Split to : ", fileName)
		checksum := sha256.Sum256(partBuffer)
		metafile = append(metafile, checksum[:]...)
	}

	metahash := sha256.Sum256(metafile)
	fmt.Println(metahash)

	metadata := &Metadata{
		Filename: filename,
		FileSize: fileSize,
		MetaFile: metafile, // concatenation of the hash of the chunks
		MetaHash: metahash, // unique identifier for the file (hash of the metafile)
	}

	return metadata
}