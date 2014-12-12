package main

import (
	"encoding/base64"
	"fmt"
	api "github.com/armon/consul-api"
	"strings"
)

type Consul struct {
	Client   *api.Client
	KVPrefix string
}

type ServiceList []*Service

type Service struct {
	Name       string
	MountPoint string
	Nodes      []*api.Node
	Port       int
}

type Node struct {
	Hostname string
	Port     int
}

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

// Takes the key from a consul KVPair from consul and strips off the prefix
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

// Does the actual query to Consul for the service names underneath the
// KVPrefix
// TODO: Allow for blocking queries
func (c *Consul) GetListOfServices() (*ServiceList, error) {
	kvs, _, err := c.Client.KV().List(c.KVPrefix, nil)
	if err != nil {
		return nil, err
	}
	return c.MapKVPairsToServiceList(kvs), nil
}

// Takes a slice of consul KVPairs and returns a ServiceList
func (c *Consul) MapKVPairsToServiceList(kvs api.KVPairs) *ServiceList {
	list := make(ServiceList, len(kvs), len(kvs))
	for i, kv := range kvs {
		list[i] = c.MapKVToService(kv)
	}
	return &list
}

// Loops over the Consul Health Service data and adds the nodes and Port to the
// Service
func (c *Consul) AddNodesToService(service *Service, serviceHealth []*api.ServiceEntry) *Service {
	length := len(serviceHealth)
	service.Port = serviceHealth[1].Service.Port
	service.Nodes = make([]*api.Node, length, length)
	for i, s := range serviceHealth {
		service.Nodes[i] = s.Node
	}
	return service
}

// Does the actual query to Consul and adds the Healthy Nodes to the service
// TODO: Allow for blocking queries
func (c *Consul) GetHealthyNodesForService(service *Service) (*Service, error) {
	healthyServices, _, err := c.Client.Health().Service(service.Name, "", true, nil)
	if err != nil {
		return nil, err
	}
	c.AddNodesToService(service, healthyServices)
	return service, nil
}

// Gets all the nodes from Consul
func (c *Consul) GetAllHealthyNodes(serviceList *ServiceList) (*ServiceList, error){
  for _, s := range(*serviceList) {
    _, err := c.GetHealthyNodesForService(s)
    if err != nil {
      return serviceList, err
    }
  }
  return serviceList, nil
}
