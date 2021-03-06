package main

import (
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/consul/api"
	"strings"
)

// Consul holds the consul configuration
type Consul struct {
	Client   *api.Client
	KVPrefix string
}

// Service is our internal mapping for a service
type Service struct {
	Name       string
	MountPoint string
	Nodes      []Node
}

// ServiceList is just an array of services
type ServiceList []*Service

// Node is the representation of a Service running on a Server
type Node struct {
	Name    string
	Address string
	Port    int
}

// NewConsul returns a new Consul object given a URL, datacenter and KV prefix
func NewConsul(address, datacenter, kvprefix string) (*Consul, error) {
	config := api.DefaultConfig()
	config.Address = address
	if datacenter != "" {
		config.Datacenter = datacenter
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Consul{KVPrefix: kvprefix, Client: client}, nil
}

// CleanupServiceName takes the key from a consul KVPair from consul and strips
// off the KVPrefix.
func (c *Consul) CleanupServiceName(name string) string {
	return strings.TrimPrefix(name, fmt.Sprintf("%s/", c.KVPrefix))
}

// Takes a consul KVPair and returns a Service struct
func (c *Consul) MapKVToService(kv *api.KVPair) *Service {
	mount, err := base64.StdEncoding.DecodeString(string(kv.Value))
	name := c.CleanupServiceName(kv.Key)
	if err != nil {
		return &Service{
			Name:       name,
			MountPoint: fmt.Sprintf("/%s", name),
		}
	}
	return &Service{
		Name:       name,
		MountPoint: string(mount),
	}
}

// GetListOfServices does the actual query to Consul for the service names
// underneath the KVPrefix
func (c *Consul) GetListOfServices() (*ServiceList, error) {
	kvs, _, err := c.Client.KV().List(c.KVPrefix, nil)
	if err != nil {
		return nil, err
	}
	return c.MapKVPairsToServiceList(kvs), nil
}

// MapKVPairsToServiceList takes a slice of consul KVPairs and returns a ServiceList
func (c *Consul) MapKVPairsToServiceList(kvs api.KVPairs) *ServiceList {
	list := make(ServiceList, len(kvs), len(kvs))
	for i, kv := range kvs {
		list[i] = c.MapKVToService(kv)
	}
	return &list
}

// AddNodesToService Loops over the Consul Health Service data and adds the nodes
// and Port to the Service
func (c *Consul) AddNodesToService(service *Service, serviceHealth []*api.ServiceEntry) *Service {
	length := len(serviceHealth)
	if length < 1 {
		return service
	}
	service.Nodes = make([]Node, length, length)
	for i, s := range serviceHealth {
		n := *s.Node
		sv := *s.Service
		service.Nodes[i] = Node{Name: n.Node, Address: n.Address, Port: sv.Port}
	}
	return service
}

// GetHealthyNodesForService Does the actual query to Consul and adds the Healthy
// Nodes to the service
func (c *Consul) GetHealthyNodesForService(service *Service) (*Service, error) {
	healthyServices, _, err := c.Client.Health().Service(service.Name, "", true, nil)
	if err != nil {
		return nil, err
	}
	c.AddNodesToService(service, healthyServices)
	return service, nil
}

// GetAllHealthyNodes Gets all the healthy nodes for each service from Consul
func (c *Consul) GetAllHealthyNodes(serviceList *ServiceList) (*ServiceList, error) {
	for _, s := range *serviceList {
		_, err := c.GetHealthyNodesForService(s)
		if err != nil {
			return serviceList, err
		}
	}
	return serviceList, nil
}
