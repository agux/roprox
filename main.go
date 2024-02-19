package main

import (
	"sync"

	"github.com/agux/roprox/conf"
	"github.com/agux/roprox/logging"
	"github.com/sirupsen/logrus"
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

	log.Infof("config file used: %s", conf.ConfigFileUsed())
	log.Infof("roprox starting on port %d", conf.Args.Proxy.Port)

	var wg sync.WaitGroup

	wg.Add(3)
	go scan(&wg)
	go serve(&wg)
	go check(&wg)

	wg.Wait()
}
