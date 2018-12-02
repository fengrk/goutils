package common

import (
	"crypto/md5"
	"fmt"
	"github.com/axgle/mahonia"
	"github.com/satori/go.uuid"
	"io"
	"math"
	"os"
)

var encodingLists = []string{"utf-8", "gb18030"}

func GetUUID() (content string) {
	u1 := uuid.Must(uuid.NewV4())
	return u1.String()
}

func GetMd5(content string) string {
	md5Writer := md5.New()
	io.WriteString(md5Writer, content)
	return fmt.Sprintf("%x", md5Writer.Sum(nil))
}

// ref: https://www.golangnote.com/topic/39.html
func GetMd5FromFile(filePath string) (string, error) {
	const fileChunk = 8192 // we settle for 8KB
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// calculate the file size
	info, _ := file.Stat()
	fileSize := info.Size()
	blocks := uint64(math.Ceil(float64(fileSize) / float64(fileChunk)))
	hash := md5.New()
	for i := uint64(0); i < blocks; i++ {
		blockSize := int(math.Min(fileChunk, float64(fileSize-int64(i*fileChunk))))
		buf := make([]byte, blockSize)
		file.Read(buf)
		io.WriteString(hash, string(buf)) // append into the hash
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func SmartEncoder(raw []byte, encoding string) (content string, err error) {
	rawStr := string(raw)
	if len(encoding) > 0 {
		return mahonia.NewDecoder(encoding).ConvertString(string(rawStr)), nil
	}
	
	for _, encoding := range encodingLists {
		// todo no error when using wrong encoding
		return mahonia.NewEncoder(encoding).ConvertString(rawStr), nil
	}
	return rawStr, nil
}
