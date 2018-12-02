package profileutils

import (
	"github.com/frkhit/logger"
	"os"
	"runtime/pprof"
)

func GolangProfile() {
	// todo just demo, not finish now
	logger.Warningln("`GolangProfile` not work well, ignore this method now!")
	return
	
	cpuFile, err := os.Create("cpu.prof.log")
	if err != nil {
		pprof.StartCPUProfile(cpuFile)
		pprof.StopCPUProfile()
		cpuFile.Close()
	}
	
	memFile, memErr := os.Create("mem.prof.log")
	if memErr != nil {
		pprof.WriteHeapProfile(memFile)
		memFile.Close()
	}
}
