package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/axeal/podwatch/internal/controller"
)

func main() {
	controller, _ := controller.NewController()
	go controller.Start()

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
	controller.Stop()
	os.Exit(0)
}
