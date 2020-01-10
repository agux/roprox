package main

import (
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/carusyte/roprox/conf"
	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	logFormatter := new(logrus.TextFormatter)
	logFormatter.FullTimestamp = true
	logFormatter.TimestampFormat = "2006-01-02 15:04:05"
	logrus.SetFormatter(logFormatter)

	logFile, _ := os.OpenFile(conf.Args.Logging.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	logrus.SetOutput(io.MultiWriter(os.Stderr, logFile))
	logrus.RegisterExitHandler(func() {
		if logFile != nil {
			logFile.Close()
		}
	})
}
