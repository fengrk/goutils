package executils

import (
	"github.com/mitchellh/go-ps"
	"strings"
)

func ProcessIsRunning(processName string) bool {
	processList, err := ps.Processes()
	if err == nil {
		for _, process := range processList {
			if strings.Index(process.Executable(), processName) > -1 {
				return true
			}
		}
	}
	return false
}
