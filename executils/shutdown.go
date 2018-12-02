package executils

import (
	"github.com/frkhit/logger"
	"os"
	"os/signal"
	"syscall"
)

func ShutdownGracefully(cleanup func()) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Infoln("got exit signal, trying to exist now...")
		cleanup()
		os.Exit(1)
	}()
}
