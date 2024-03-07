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

	var wg sync.WaitGroup

	if conf.Args.Scanner.Enabled {
		log.Infof("starting scanner")
		wg.Add(1)
		go scanner.Scan(&wg)
	}
	if conf.Args.Proxy.Enabled {
		log.Infof("starting proxy on port %d", conf.Args.Proxy.Port)
		wg.Add(1)
		go proxy.Serve(&wg)
	}
	if conf.Args.Probe.Enabled {
		log.Infof("starting probe")
		wg.Add(1)
		go checker.Check(&wg)
	}

	wg.Wait()
}
