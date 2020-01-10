package main

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	logFormatter := new(logrus.TextFormatter)
	logFormatter.FullTimestamp = true
	logFormatter.TimestampFormat = "2006-01-02 15:04:05"
	logrus.SetFormatter(logFormatter)
}
