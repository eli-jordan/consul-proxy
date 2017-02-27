## ConsulProxy

A TCP proxy that dynamically updates the backend service, based on consul service discovery lookups.

### Usage

```
consul-proxy -h
Usage of consul-proxy:
  -config-file string
        The fully qualified path the json configuration file specifying the services to proxy
  -consul-dns-name string
        The DNS name used to lookup the consul server
  -consul-server-override string
        The host:port where the consul ReST API that should be used for discovery is running
  -dns-port string
        The port used when making a DNS query to the specified DNS server
  -dns-server string
        The DNS server that is used to discover consul
  -service value
        The consul services to proxy in the format :{port-on-localhost}/{service-name}. This flag can be specified multiple times to proxy multiple services.

```

### Configuration

The tool can be configured using command-line arguments, a json configuration file, or a combination of both with command line arguments overriding the json configuration file.

**Finding The Consul Server**

This tool makes requests to the consul ReST API to discover the service instances that are being proxied. There are two way that the consul server can be specified.

1. *Specify a static host*
	* The host/port that are given are used directly
	* Use the `-consul-server-override` command line argument, or the `ConsulServer.Address` attribute in the config file.
	
2. *DNS SRV Lookup*
	* The consul server is lookup up by making a DNS SRV query for the specified name to the specified DNS server
	* Use the `-dns-server` and `-dns-port` command line arguments, or the `ConsulServer.DnsServer` and `ConsulServer.DnsPort` attributes in the config file to specify the DNS server that is used.
	* Use the `-consul-dns-name` or the `ConsulServer.DnsName` to specify the name used with the SRV query

#### Example JSON Config
```
{
  "ConsulServer": {
    "DnsServer": "123.123.123.123",
    "DnsName": "prod-infra-rtp-consul-external.query.ibm"
  },
  "Proxies": [
    {
      "ServiceName": "my-service-name",
      "LocalPort": 9090
    }
  ]
}
```

### TODO

Load balancing options for the backend services