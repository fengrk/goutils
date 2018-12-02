package logger

import (
	"github.com/frkhit/logger"
	"os"
)

const logPath = ""

var defaultLogger *logger.Logger = nil
var logFile *os.File = nil

func init() {
	CreateLogger()
}

func CreateLogger() {
	if defaultLogger != nil {
		return
	}
	
	var err error
	if len(logPath) > 0 {
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
		if err != nil {
			logger.Fatalf("Failed to open log file: %v", err)
			os.Exit(1)
		}
		defaultLogger = logger.Init("LoggerExample", true, false, logFile)
		return
	}
	defaultLogger = logger.Init("LoggerExample", true, false, nil)
}

func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
	if defaultLogger != nil {
		defaultLogger.Close()
	}
}
