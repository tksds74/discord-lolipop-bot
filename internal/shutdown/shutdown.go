package shutdown

import (
	"os"
	"os/signal"
	"syscall"
)

func WaitForExitSignal() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-signalChannel
}
