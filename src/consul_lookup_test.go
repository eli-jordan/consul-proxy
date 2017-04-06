package main

import (
	"testing"
	"errors"
	consul "github.com/hashicorp/consul/api"
	"time"
)

func TestConsulLookup_getConsulServer_OverrideAddress(t *testing.T) {
	config := &ConsulServerConfig {
		Address: "this.is.an.override.address",
	}
	lookup := NewConsulLookup("test-service-name", "", config)
	result, err := lookup.getConsulServer()

	assertNil(t, err)
	assertEqual(t, "this.is.an.override.address", result, "ConsulServer")
}

func TestConsulLookup_getConsulServer_SrvLookup(t *testing.T) {
	config := &ConsulServerConfig{}
	lookup := NewConsulLookup("test-service-name", "", config)
	lookup.dnsSrv = stubSrvLookup("1.2.3.4:1234", nil)
	result, err := lookup.getConsulServer()

	assertNil(t, err)
	assertEqual(t, "1.2.3.4:1234", result, "ConsulServer")
}

func TestConsulLookup_getConsulServer_SrvLookup_Error(t *testing.T) {
	config := &ConsulServerConfig{}
	lookup := NewConsulLookup("test-service-name", "", config)
	lookup.dnsSrv = stubSrvLookup("", errors.New("this.is.an.errors"))
	_, err := lookup.getConsulServer()

	assertNotNil(t, err)
	assertEqual(t, err.Error(), "this.is.an.errors", "ConsulServer")
}

func TestConsulLookup_lookup(t *testing.T) {
	config := &ConsulServerConfig {
		Address: "this.is.an.override.address",
	}
	lookup := NewConsulLookup("test-service-name", "", config)

	entry := &consul.ServiceEntry{
		Service: &consul.AgentService {
			Address: "an-address",
			Port: 1234,
		},
	}
	lookup.consulRest = stubConsulRestLookup([]*consul.ServiceEntry{entry}, nil)

	endpoints, err := lookup.lookup()

	assertNil(t, err)
	assertEqual(t, len(endpoints), 1, "len(endpoints)")
	assertEqual(t, endpoints[0].host, "an-address", "endpoint hostname")
	assertEqual(t, endpoints[0].port, 1234, "endpoint port")
}

func TestConsulLookup_start(t *testing.T) {
	config := &ConsulServerConfig {
		Address: "this.is.an.override.address",
	}
	lookup := NewConsulLookup("test-service-name", "", config)
	lookup.pollIntervalSec = 1

	entry1 := &consul.ServiceEntry{
		Service: &consul.AgentService {
			Address: "an-address-1",
			Port: 1234,
		},
	}

	entry2 := &consul.ServiceEntry{
		Service: &consul.AgentService {
			Address: "an-address-2",
			Port: 4567,
		},
	}

	lookup.consulRest = stubConsulRestLookup([]*consul.ServiceEntry{entry1}, nil)
	lookup.start()

	endpoints1 := lookup.getEndpoints()
	assertEqual(t, endpoints1[0].host, "an-address-1", "first lookup")

	lookup.consulRest = stubConsulRestLookup([]*consul.ServiceEntry{entry2}, nil)
	time.Sleep(2 * time.Second)

	endpoints2 := lookup.getEndpoints()
	assertEqual(t, endpoints2[0].host, "an-address-2", "second lookup")

}

func stubSrvLookup(result string, err error) DnsSrvLookup {
	return func(dnsServer string, dnsPort string, name string) (string, error) {
		return result, err
	}
}

func stubConsulRestLookup(services []*consul.ServiceEntry, err error) ConsulRestLookup {
	return func(consulAddress string, serviceName string, datacenter string) ([]*consul.ServiceEntry, error) {
		return services, err
	}
}
