package main

import (
	"sync"

	"github.com/agux/roprox/internal/checker"
	"github.com/agux/roprox/internal/conf"
	"github.com/agux/roprox/internal/logging"
	"github.com/agux/roprox/internal/proxy"
	"github.com/agux/roprox/internal/scanner"
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
	go scanner.Scan(&wg)
	go proxy.Serve(&wg)
	go checker.Check(&wg)

	wg.Wait()
}
