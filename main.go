package main

import (
	"sync"

	"github.com/carusyte/roprox/logging"
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

	log.Info("roprox starting...")

	var wg sync.WaitGroup
	wg.Add(2)

	go scan(&wg)
	go check(&wg)

	wg.Wait()
}
