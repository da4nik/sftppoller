package main

import (
	"context"
	"os"
	"os/signal"

	log "github.com/Sirupsen/logrus"
	"github.com/da4nik/sftppoller/config"
	"github.com/da4nik/sftppoller/poller"
	_ "github.com/joho/godotenv/autoload"
)

var logFile *os.File

func initLogger() {
	// Setting up logger
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)

	log.SetOutput(os.Stdout)
	if len(config.LogFileName) > 0 {
		var err error
		logFile, err = os.OpenFile(config.LogFileName, os.O_WRONLY|os.O_CREATE, 0664)
		if err != nil {
			log.Warningf("File %s, can't be opened, using STDOUT for logging.", config.LogFileName)
		} else {
			log.SetOutput(logFile)
		}
	}
}

func main() {
	config.Load()
	initLogger()

	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	poller.Start(pollerCtx)
	defer pollerCancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
