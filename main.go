package main

import (
	"sync"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Info("roprox starting...")
	defer logrus.Exit(0)
	
	var wg sync.WaitGroup
	wg.Add(2)

	go scan(&wg)
	go check(&wg)

	wg.Wait()
}
