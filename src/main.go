package main

import (
	"log"
	"fmt"
	"sync"
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
		lookup := NewConsulLookup(conf.ServiceName, configuration.ConsulServer)
		proxy := NewConsulProxy(conf, lookup)
		go func() {
			defer wg.Done()
			proxy.start()
		}()
	}

	wg.Wait()
}