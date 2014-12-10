package main

import (
  api "github.com/armon/consul-api"
  "strings"
  "fmt"
  "encoding/base64"
)

type Consul struct {
  Client *api.Client
  KVPrefix string
}

type ServiceList []*Service

type Service struct {
  Name string
  MountPoint string
  Nodes []*Node
}

type Node struct {
  Hostname string
  Port int
}

func NewConsul(address string, datacenter string) (*Consul, error) {
  config := api.DefaultConfig()
  config.Address = address
  if datacenter != "" {
    config.Datacenter = datacenter
  }

  client, err := api.NewClient(config)
  if err != nil {
    return &Consul{}, err
  }

  return &Consul{KVPrefix: "conductor-services", Client: client}, nil
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
      Name: name,
      MountPoint: fmt.Sprintf("/%s", name),
    }
  }
  return &Service{
    Name: name,
    MountPoint: string(mount),
  }
}

// Takes a slice of consul KVPairs and returns a ServiceList
func (c *Consul) MapKVPairsToServiceList(kvs *api.KVPairs) *ServiceList {
  list := make(ServiceList, len(*kvs), len(*kvs))
  for i, kv := range(*kvs) {
    list[i] = c.MapKVToService(kv)
  }
  return &list
}
