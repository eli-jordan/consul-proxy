package main

import (
	"net"
	"strconv"
	"github.com/miekg/dns"
	consul "github.com/hashicorp/consul/api"
	"log"
	"time"
	"sync"
)

/**
 * This file contains the data types and logic involved in discovering a consul service
 */

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

// Abstracts the dns srv lookup used to discover the consul server
type DnsSrvLookup func(
	/* dnsSever */ string,
	/* dnsPort  */ string,
	/* name     */ string) (string, error)

// Abstracts the invocation of the consul ReST API
// to lookup a service by its name
type ConsulRestLookup func(
	/* consulAddress */ string,
	/* serviceName   */ string,
    /* datacenter    */ string) ([]*consul.ServiceEntry, error)

/**
 * Contains the dynamically updating endpoints associated with the provides
 * service. The endpoints are updated in the background, on the specified schedule.
 */
type ConsulLookup struct {
	// the name of the service associated with this lookup instance
	serviceName  string

	// the datacenter the service should be looked up in
	datacenter string

	// the consul server that ReST API calls are made against
	consulServer *ConsulServerConfig

	// the current set of endpoints associated with the service
	// must be accessed under endpointsMu
	endpoints    []*Endpoint
	endpointsMu  sync.Mutex

	dnsSrv       DnsSrvLookup
	consulRest   ConsulRestLookup

	// How often to poll consul for the service addresses
	pollIntervalSec time.Duration
}

/**
 * serviceName - the consul service name to discover
 * consulServer - the config used to lookup the consul server to make ReST requests to
 */
func NewConsulLookup(serviceName string, datacenter string, consulServer *ConsulServerConfig) *ConsulLookup {
	return &ConsulLookup{
		serviceName: serviceName,
		datacenter: datacenter,
		consulServer: consulServer,
		pollIntervalSec: 30,
		dnsSrv: dnsSrvLookup,
		consulRest: consulRestLookup,
	}
}

/**
 * Starts the lookup process that periodically discovers the configured
 * services in consul, so that new TCP connections can be established
 * using an up to data backend.
 */
func (cl *ConsulLookup) start() {
	var closed = false
	done := make(chan struct{})
	go func() {
		for range time.NewTicker(cl.pollIntervalSec * time.Second).C {
			endpoints, err := cl.lookup()
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

/**
 * Read the current backend endpoints using the appropriate lock
 */
func (cl *ConsulLookup) getEndpoints() []*Endpoint {
	cl.endpointsMu.Lock()
	defer cl.endpointsMu.Unlock()

	result := cl.endpoints
	return result
}

/**
 * Performs a consul lookup based on the provided config, which is used to find the consul server,
 * then finds all healthy instances of the named service using the consul ReST API.
 */
func (cl *ConsulLookup) lookup() ([]*Endpoint, error) {

	server, err := cl.getConsulServer()
	if err != nil {
		return nil, err
	}

	services, err := cl.consulRest(server, cl.serviceName, cl.datacenter)

	endpoints := make([]*Endpoint, len(services))
	for i, s := range services {
		endpoints[i] = &Endpoint {
			host: s.Service.Address,
			port: s.Service.Port,
		}
	}
	return endpoints, nil
}

/**
 * Finds the consul servers hostname/port.
 *
 * 1. If the 'Address' config is provided, it is simply used as is.
 * 2. Otherwise an SRV record is looked up using the DNS server defined by DnsServer  and DnsPort
 *    configurations. The default DnsServer=localhost default DnsPort=53
 */
func (cl *ConsulLookup) getConsulServer() (string, error) {
	if cl.consulServer.Address != "" {
		return cl.consulServer.Address, nil
	} else {
		dnsServer := cl.consulServer.DnsServer
		if dnsServer == "" {
			dnsServer = "localhost"
		}

		dnsPort := cl.consulServer.DnsPort
		if dnsPort == "" {
			dnsPort = "53"
		}

		log.Printf("Looking SRV record for %s using %s", cl.consulServer.DnsName, dnsServer + ":" + dnsPort)

		address, err := cl.dnsSrv(dnsServer, dnsPort, cl.consulServer.DnsName)
		if err != nil {
			log.Printf("Failed to execute DNS SRV lookup: %s", err)
			return "", err
		} else {
			log.Printf("Found consul server at '%s'", address)
			return address, nil
		}
	}
}

func consulRestLookup(consulAddress string, serviceName string, datacenter string) ([]*consul.ServiceEntry, error) {
	config := consul.DefaultConfig()
	config.Address = consulAddress

	log.Printf("Using consul server %s to lookup service=%s in datacenter=%s", consulAddress, serviceName, datacenter)

	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}

	options := &consul.QueryOptions{
		Datacenter: datacenter,
	}

	services, _, err := client.Health().Service(serviceName, "", true, options)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func dnsSrvLookup(dnsServer string, dnsPort string, name string) (string, error) {

	clientConfig := &dns.ClientConfig {
		Servers: []string { dnsServer },
		Port: dnsPort,
	}
	results, err := lookupSrv(name, clientConfig)
	if err != nil {
		return "", err
	}

	return results[0], nil
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