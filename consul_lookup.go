package main

import (
	"net"
	"strconv"
	"github.com/miekg/dns"
	"github.com/hashicorp/consul/api"
	"log"
	"time"
	"sync"
)

/**
 * Represents a TCP endpoint
 */
type Endpoint struct {
	host string
	port int
}

func (ep *Endpoint) String() string {
	return (ep.host + ":" + strconv.Itoa(ep.port))
}

/**
 * Contains the dynamically updating endpoints associated with the provides
 * service. The endpoints are updated in the background, on the specified schedule.
 */
type ConsulLookup struct {
	// the name of the service associated with this lookup instance
	serviceName string

	// the consul server that ReST API calls are made against
	consulServer *ConsulServerConfig

	// the current set of endpoints associated with the service
	// must be accessed under endpointsMu
	endpoints   []*Endpoint
	endpointsMu sync.Mutex
}

/**
 * serviceName - the consul service name to discover
 * consulServer - (optional)
 */
func NewConsulLookup(serviceName string, consulServer *ConsulServerConfig) *ConsulLookup {
	return &ConsulLookup{
		serviceName: serviceName,
		consulServer: consulServer,
	}
}

func (cl *ConsulLookup) start() {
	var closed = false
	done := make(chan struct{})
	go func() {
		for range time.NewTicker(10 * time.Second).C {
			endpoints, err := lookup(cl.consulServer, cl.serviceName)
			if err != nil {
				log.Printf("Error discovering service %s - %s", cl.serviceName, err)
				continue
			}

			log.Printf("Discovered services %s", endpoints)

			if !closed {
				close(done)
				closed = true
			}

			cl.endpointsMu.Lock()
			cl.endpoints = endpoints
			cl.endpointsMu.Unlock()
		}
	}()

	<-done
}

func (cl *ConsulLookup) getEndpoints() []*Endpoint {
	cl.endpointsMu.Lock()
	result := cl.endpoints
	cl.endpointsMu.Unlock()
	return result
}

func lookup(consulServer *ConsulServerConfig, service string) ([]*Endpoint, error) {

	server, err := getConsulServer(consulServer)
	if err != nil {
		return nil, err
	}

	config := api.DefaultConfig()
	config.Address = server
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	services, _, err := client.Health().Service(service, "", true, nil)
	if err != nil {
		return nil, err
	}

	endpoints := make([]*Endpoint, len(services))
	for i, s := range services {
		endpoints[i] = &Endpoint{
			host: s.Service.Address,
			port: s.Service.Port,
		}
	}
	return endpoints, nil

}

func getConsulServer(config *ConsulServerConfig) (string, error) {
	if config.Address != "" {
		return config.Address, nil
	} else {
		dnsServer := config.DnsServer
		if dnsServer == "" {
			dnsServer = "localhost"
		}

		dnsPort := config.DnsPort
		if dnsPort == "" {
			dnsPort = "53"
		}

		log.Printf("Looking SRV record for %s using %s", config.DnsName, dnsServer + ":" + dnsPort)

		clientConfig := &dns.ClientConfig{
			Servers: []string{ dnsServer },
			Port: dnsPort,
		}

		address, err := lookupSrv(config.DnsName, clientConfig)
		if err != nil {
			log.Printf("Failed to execute DNS SRV lookup: %s", err)
			return "", err
		} else {
			resultAddress := address[0]
			log.Printf("Found consul server at '%s'", resultAddress)
			return resultAddress, nil
		}
	}
}



/**
 * Looks up the specified domain name using an SRV DNS query to the server(s) specified
 * in the client config.
 *
 * Returns all DNS answers in a host:port format
 */
func lookupSrv(address string, config *dns.ClientConfig) ([]string, error) {
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn(address), dns.TypeSRV)
	query.RecursionDesired = false
	client := new(dns.Client)
	resp, _, err := client.Exchange(query, net.JoinHostPort(config.Servers[0], config.Port))
	if err != nil {
		return nil, err
	}
	if len(resp.Answer) == 0 {
		return []string{}, nil
	}
	var addrs []string
	for i, record := range resp.Answer {
		port := strconv.Itoa(int(record.(*dns.SRV).Port))
		ip := record.(*dns.SRV).Target
		if len(resp.Extra) >= i + 1 {
			ip = resp.Extra[i].(*dns.A).A.String()
		}
		addrs = append(addrs, net.JoinHostPort(ip, port))
	}
	return addrs, nil
}
