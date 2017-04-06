package main

import (
	"testing"
)

func TestParseJsonDefiningAllPossibleFields(t *testing.T) {
	config := parseConfig([]byte(
			`{
				"ConsulServer": {
				   "DnsServer": "123.123.123.123",
				   "DnsPort": "123",
				   "DnsName": "prod-infra-rtp-consul-external.query.ibm",
				   "Address": "this-is-an-address-override.com"
				},
				"Proxies": [
				   {
					  "ServiceName": "service-a",
					  "Datacenter": "foo-bar",
					  "LocalIP": "0.0.0.0",
					  "LocalPort": 9090
				   },
				   {
					  "ServiceName": "service-b",
					  "LocalIP": "0.0.0.0",
					  "LocalPort": 9091
				   }
				]
			 }`))

	assertEqual(t, "123.123.123.123", config.ConsulServer.DnsServer, "DnsServer")
	assertEqual(t, "123", config.ConsulServer.DnsPort, "DnsPort")
	assertEqual(t, "prod-infra-rtp-consul-external.query.ibm", config.ConsulServer.DnsName, "DnsName")
	assertEqual(t, "this-is-an-address-override.com", config.ConsulServer.Address, "Address")
	assertEqual(t, "service-a", config.Proxies[0].ServiceName, "Proxies[0].ServiceName")
	assertEqual(t, "0.0.0.0", config.Proxies[0].LocalIP, "Proxies[0].LocalIP")
	assertEqual(t, 9090, config.Proxies[0].LocalPort, "Proxies[0].LocalPort")
	assertEqual(t, "foo-bar", config.Proxies[0].Datacenter, "Proxies[1].Datacenter")
	assertEqual(t, "", config.Proxies[1].Datacenter, "Empty Datacenter")
}

func TestInterpretCommandLine_Simple(t *testing.T) {
	args := CliArgs{
		configFile: "./test_config.json",
		consulDnsName: "prod-infra-rtp-consul-external.query.ibm",
		dnsServer: "1.2.3.4",
		dnsPort: "1234",
	}

	config, _ := interpretCommandLine(&args)
	assertEqual(t, "1.2.3.4", config.ConsulServer.DnsServer, "DnsServer")
	assertEqual(t, "1234", config.ConsulServer.DnsPort, "DnsPort")
	assertEqual(t, "prod-infra-rtp-consul-external.query.ibm", config.ConsulServer.DnsName, "DnsName")
}

func TestInterpretCommandLine_NoServices(t *testing.T) {
	args := CliArgs{
		consulDnsName: "prod-infra-rtp-consul-external.query.ibm",
		dnsServer: "1.2.3.4",
		dnsPort: "1234",
	}

	_, err := interpretCommandLine(&args)

	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestInterpretCommandLine_CliServicesAndConfigFile(t *testing.T) {
	args := CliArgs{
		configFile: "./test_config.json",
		services: ProxiedServiceList{
			values: []*ProxiedService{
				{
					ServiceName: "foo",
				},
			},
		},
	}

	_, err := interpretCommandLine(&args)

	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestProxiedServiceList_Set_NoDatacenter(t *testing.T) {
	// common format :{port}/{service-name}
	list := &ProxiedServiceList{}
	err1 := list.Set(":9092/my-service-name1")

	assertNil(t, err1)
	assertEqual(t, "localhost", list.values[0].LocalIP, "LocalIP")
	assertEqual(t, "my-service-name1", list.values[0].ServiceName, "ServiceName")
	assertEqual(t, 9092, list.values[0].LocalPort, "LocalPort")
}

func TestProxiedServiceList_Set_WithDatacenter(t *testing.T) {
	// common format :{port}/{service-name}
	list := &ProxiedServiceList{}
	err1 := list.Set(":1111/my-service-name2/sydney")

	assertNil(t, err1)
	assertEqual(t, "localhost", list.values[0].LocalIP, "LocalIP")
	assertEqual(t, "my-service-name2", list.values[0].ServiceName, "ServiceName")
	assertEqual(t, 1111, list.values[0].LocalPort, "LocalPort")
	assertEqual(t, "sydney", list.values[0].Datacenter, "Datacenter")
}

func TestProxiedServiceList_Set_WithBindPort(t *testing.T) {
	// specify bind ip {bind-ip}:{port}/{service-name}
	list := &ProxiedServiceList{}
	err2 := list.Set("0.0.0.0:9092/my-service-name2")
	assertNil(t, err2)
	assertEqual(t, "0.0.0.0", list.values[0].LocalIP, "LocalIP")
	assertEqual(t, "my-service-name2", list.values[0].ServiceName, "ServiceName")
	assertEqual(t, 9092, list.values[0].LocalPort, "ServiceName")
}


func assertEqual(t *testing.T, expected interface{}, actual interface{}, message string) {
	if expected != actual {
		t.Fatal(message, "Expected:", expected, "Actual:", actual)
	}
}

func assertNil(t *testing.T, value interface{}) {
	if value != nil {
		t.Fatalf("Expected nil %v", value)
	}
}

func assertNotNil(t *testing.T, value interface{}) {
	if value == nil {
		t.Fatal("Expected not nil")
	}
}



