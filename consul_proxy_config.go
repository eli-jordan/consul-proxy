package main

import (
	"strconv"
	"fmt"
	"io/ioutil"
	"os"
	"encoding/json"
	"log"
	"flag"
	"strings"
)

/**
 * Represents a single proxied service
 */
type ProxiedService struct {
	// the consul service name used to discover the service
	ServiceName string

	// the ip for the frontend to bind to - defaults to localhost
	LocalIP     string

	// the port for the frontend to bind to
	LocalPort   int
}

func (ps *ProxiedService) String() string {
	return "localhost:" + strconv.Itoa(ps.LocalPort) + " -> Consul(" + ps.ServiceName + ")"
}

/**
 * The config options used to control how the consul rest server is discovered.
 *
 * If the 'Address' is specified, this will simply be used as the ReST endpoint.
 *
 * Otherwise, 'DnsName' is used to lookup an SRV record against the DNS servers
 * specified in 'DnsServers'
 *
 * If 'DnsServers' is not specified 'localhost:53' is used.
 */
type ConsulServerConfig struct {
	// the DNS servers used to lookup the consul server
	DnsServer string
	DnsPort   string

	// the DNS name used to lookup the consul server
	DnsName   string

	// the override address for the consul server
	Address   string
}

/**
 * Represents the full configuration for a consul proxy instance
 */
type ConsulProxyConfig struct {
	// The config options that specify how the consumer server
	// should be accessed.
	ConsulServer *ConsulServerConfig

	// The list of services that should be proxied and what local port should be bound.
	Proxies      []*ProxiedService
}

func (cpc *ConsulProxyConfig) String() string {
	return fmt.Sprint("DnsServers: ", cpc.ConsulServer.DnsServer, ", Consul Server: ", cpc.ConsulServer.Address, ", Proxies: ", cpc.Proxies)
}


/**
 * Parses the configuration json file
 */
func readJson(file string) *ConsulProxyConfig {
	data, readErr := ioutil.ReadFile(file)
	if readErr != nil {
		log.Fatal("Error reading comfig file", file, readErr)
		os.Exit(1)
	}

	var config ConsulProxyConfig
	marshalErr := json.Unmarshal(data, &config)
	if marshalErr != nil {
		log.Fatal("Error reading comfig file", file, marshalErr)
		os.Exit(1)
	}
	return &config
}

/**
 * A helper type that implements the flags.Var interface, that enables
 * parsing of multiple -service flags
 */
type ProxiedServiceList struct {
	values []*ProxiedService
}

// Parses a ProxiedService from the specified string
func (v *ProxiedServiceList) Set(value string) error {
	proxied := strings.Split(value, "/")
	if len(proxied) != 2 {
		log.Println("Proxied service", proxied, "has an invalid format")
		os.Exit(1)
	}

	serviceName := proxied[1]

	var localIP string
	var localPort string
	local := strings.Split(proxied[0], ":")
	if len(local) == 1 {
		localIP = "localhost"
		localPort = local[0]
	} else if len(local) == 2 {
		localIP = local[0]
		localPort = local[1]
	}

	port, err := strconv.Atoi(localPort)
	if err != nil {
		log.Println(proxied[1], "could not be parsed as a number")
		os.Exit(1)
	}

	values := v.values

	v.values = append(values, &ProxiedService{
		ServiceName: serviceName,
		LocalIP: localIP,
		LocalPort: port,
	})

	return nil
}

func (v *ProxiedServiceList) String() string {
	return fmt.Sprintf("%d", *v)
}

/**
 * Parses the CLI arguments and resolves any config options
 */
func configuration() *ConsulProxyConfig {

	var services ProxiedServiceList
	flag.Var(&services, "service", "The consul services to proxy in the format {service-name}/:{port-on-localhost}. This flag can be specified multiple times to proxy multiple services.")

	configFile := flag.String("config-file", "", "The fully qualified path the json configuration file specifying the services to proxy")
	consulServerOverride := flag.String("consul-server-override", "", "The host:port where the consul ReST API that should be used for discovery is running")
	consulDnsName := flag.String("consul-dns-name", "", "The DNS name used to lookup the consul server")
	dnsServer := flag.String("dns-server", "", "The DNS server that is used to discover consul")
	dnsPort := flag.String("dns-port", "", "The port used when making a DNS query to the specified DNS server")

	flag.Parse()

	if *configFile != "" && len(services.values) != 0 {
		log.Println("-config-file and -proxy-services cannot both be specified. Please use one or the other.")
		os.Exit(1)
	}

	if *consulServerOverride != "" {
		log.Println("Consul server has been overriden using configuration", *consulServerOverride)
	}

	if len(services.values) == 0 && *configFile == "" {
		log.Println("No proxied services specified. Please either specify -proxy-services or -config-file")
		os.Exit(1)
	}

	// TODO: validate the consul dns lookup parameters

	config := new(ConsulProxyConfig)
	config.ConsulServer = new(ConsulServerConfig)
	if *configFile != "" {
		config = readJson(*configFile)
	}

	if *dnsServer != "" {
		config.ConsulServer.DnsServer = *dnsServer
	}

	if *dnsPort != "" {
		config.ConsulServer.DnsPort = *dnsPort
	}

	if *consulDnsName != "" {
		config.ConsulServer.DnsName = *consulDnsName
	}

	if *consulServerOverride != "" {
		config.ConsulServer.Address = *consulServerOverride
	}

	if len(services.values) != 0 {
		config.Proxies = services.values
	}

	return config
}