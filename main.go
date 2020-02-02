package main

import (
	"sync"

	"github.com/carusyte/roprox/logging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logging.Logger

func main() {
	defer func() {
		code := 0
		if r := recover(); r != nil {
			if _, hasError := r.(error); hasError {
				code = 1
			}
		}
		logrus.Exit(code)
	}()

	log.Info("roprox starting...")
	log.Infof("config file used: %s", viper.ConfigFileUsed())

	var wg sync.WaitGroup
	wg.Add(2)

	go scan(&wg)
	go check(&wg)

	wg.Wait()
}
