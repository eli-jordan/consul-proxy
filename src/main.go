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

	configuration := configuration()
	log.Println("Effective Configuration", configuration)

	var wg sync.WaitGroup

	for _, proxy := range configuration.Proxies {
		wg.Add(1)
		lookup := NewConsulLookup(proxy.ServiceName, proxy.Datacenter, configuration.ConsulServer)
		proxy := NewConsulProxy(proxy, lookup)
		go func() {
			defer wg.Done()
			proxy.start()
		}()
	}

	fmt.Printf("Version: %s, Build: %s\n", Version, Build)

	wg.Wait()
}