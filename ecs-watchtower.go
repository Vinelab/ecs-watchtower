package main

import (
	"time"

	"github.com/vinelab/watchtower/infrastructure"
	"github.com/vinelab/watchtower/status"
)

func fetchAndCheck() {
	i := infrastructure.New()

	clusters := i.Clusters()
	status.Check(i, clusters)
}

func main() {
	// start working
	fetchAndCheck()

	// wait a minute and then work again
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for range ticker.C {
			fetchAndCheck()
		}
	}()

	select {}
	// Watch(checkers)
}
