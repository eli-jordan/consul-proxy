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
	"errors"
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
func readJson(file string) (*ConsulProxyConfig, error) {
	data, readErr := ioutil.ReadFile(file)
	if readErr != nil {
		log.Fatal("Error reading config file", file, readErr)
		return nil, readErr
	}
	return parseConfig(data), nil
}

func parseConfig(data []byte) *ConsulProxyConfig {
	var config ConsulProxyConfig
	marshalErr := json.Unmarshal(data, &config)
	if marshalErr != nil {
		log.Fatal("Error reading config file", marshalErr)
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

	if len(local) != 2 {
		log.Println("Proxied service", proxied, "has an invalid format")
		os.Exit(1)
	}

	if local[0] == "" {
		localIP = "localhost"
		localPort = local[1]
	} else {
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

type CliArgs struct {
	services ProxiedServiceList
	configFile string
	consulServerOverride string
	consulDnsName string
	dnsServer string
	dnsPort string
}

/**
 * Parses the CLI arguments and resolves any config options
 */
func configuration() *ConsulProxyConfig {
	cli := parseCommandLine()

	config, err := interpretCommandLine(cli)
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}

	return config
}

/**
 * Defines the command line args, and simply assigns the specified values
 * to the CliArgs struct.
 */
func parseCommandLine() *CliArgs {
	flag.Parse()

	var args CliArgs

	flag.Var(&args.services, "service", "The consul services to proxy in the format {service-name}/:{port-on-localhost}. This flag can be specified multiple times to proxy multiple services.")

	flag.StringVar(&args.configFile, "config-file", "", "The fully qualified path the json configuration file specifying the services to proxy")
	flag.StringVar(&args.consulServerOverride, "consul-server-override", "", "The host:port where the consul ReST API that should be used for discovery is running")
	flag.StringVar(&args.consulDnsName, "consul-dns-name", "", "The DNS name used to lookup the consul server")
	flag.StringVar(&args.dnsServer, "dns-server", "", "The DNS server that is used to discover consul")
	flag.StringVar(&args.dnsPort, "dns-port", "", "The port used when making a DNS query to the specified DNS server")

	return &args
}

/**
 * Reads the specified command line, and generates the final configuration
 */
func interpretCommandLine(args *CliArgs) (*ConsulProxyConfig, error) {
	if args.configFile != "" && len(args.services.values) != 0 {
		return nil, errors.New("-config-file and -proxy-services cannot both be specified. Please use one or the other.")
	}

	if args.consulServerOverride != "" {
		log.Println("Consul server has been overriden using configuration", args.consulServerOverride)
	}

	if len(args.services.values) == 0 && args.configFile == "" {
		return nil, errors.New("No proxied services specified. Please either specify -proxy-services or -config-file")
	}

	if args.consulDnsName == "" && args.consulServerOverride == "" {
		return nil, errors.New("Unable to find the consul server. Please either specify -consul-server-override or -consul-dns-name")
	}

	config := new(ConsulProxyConfig)
	config.ConsulServer = new(ConsulServerConfig)
	if args.configFile != "" {
		conf, err := readJson(args.configFile)
		if err != nil {
			return nil, err
		} else {
			config = conf
		}
	}

	if args.dnsServer != "" {
		config.ConsulServer.DnsServer = args.dnsServer
	}

	if args.dnsPort != "" {
		config.ConsulServer.DnsPort = args.dnsPort
	}

	if args.consulDnsName != "" {
		config.ConsulServer.DnsName = args.consulDnsName
	}

	if args.consulServerOverride != "" {
		config.ConsulServer.Address = args.consulServerOverride
	}

	if len(args.services.values) != 0 {
		config.Proxies = args.services.values
	}

	return config, nil
}