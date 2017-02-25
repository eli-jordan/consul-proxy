package main

import (
	"log"
	"sync"
	"fmt"
)

var (
	Version = "ConsulProxyVersion"
	Build = "ConsulProxyBuildIdentifier"
)

func main() {

	fmt.Printf("Version: %s, Build: %s", Version, Build)

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