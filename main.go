package main

import (
	"log"
	"sync"
)

func main() {

	configuration := configuration()
	log.Println("Effective Configuration", configuration)

	var wg sync.WaitGroup

	for _, conf := range configuration.Proxies {
		wg.Add(1)
		proxy := NewConsulProxy(conf, configuration.ConsulServer)
		go func() {
			defer wg.Done()
			proxy.start()
		}()
	}

	wg.Wait()
}