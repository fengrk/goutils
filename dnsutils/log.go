package dnsutils

import (
	"github.com/frkhit/goutils/common"
	"os"
	"path"
)

var logPath = "./log"

func GetLogPath(fileName string) string {
	if !common.FileExists(logPath) {
		os.Mkdir(logPath, 0600)
	}
	return path.Join(logPath, fileName)
}
