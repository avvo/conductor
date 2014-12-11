package main

import(
  "testing"
  "fmt"
  api "github.com/armon/consul-api"
)

var consul *Consul

func init() {
  consul = &Consul{KVPrefix: "conductor-services"}
}

func TestCleanupServiceName(t *testing.T) {
  expected := "solr"
  input := "conductor-services/solr"

  result := consul.CleanupServiceName(input)
  if result != expected {
    t.Error(fmt.Sprintf("%s != %s", expected, result))
  }
}

func TestMapKVToService(t *testing.T) {
  expected := &Service{Name: "solr", MountPoint: "/solr"}
  input := &api.KVPair{Key: "conductor-services/solr", Value: []byte("L3NvbHI=")}

  result := consul.MapKVToService(input)
  if result.Name != expected.Name || result.MountPoint != expected.MountPoint {
    t.Error(fmt.Sprintf("Expected:\n %v \nBut got:\n %v", expected, result))
  }
}

func TestMapKVToServiceWithInvalidBase64(t *testing.T) {
  expected := &Service{Name: "solr", MountPoint: "/solr"}
  input := &api.KVPair{Key: "conductor-services/solr", Value: []byte("!!INVALID!!")}

  result := consul.MapKVToService(input)
  if result.Name != expected.Name || result.MountPoint != expected.MountPoint {
    t.Error(fmt.Sprintf("Expected:\n %v \nBut got:\n %v", expected, result))
  }
}

func TestMapKVPairsToServiceList(t *testing.T) {
  expected := ServiceList{
    &Service{Name: "solr", MountPoint: "/solr"},
    &Service{Name: "backend_service_v1", MountPoint: "/service/v1"},
  }
  input := &api.KVPairs{
    &api.KVPair{Key: "conductor-services/solr", Value: []byte("L3NvbHI=")},
    &api.KVPair{Key: "conductor-services/backend_service_v1", Value: []byte("L3NlcnZpY2UvdjE=")},
  }

  result := consul.MapKVPairsToServiceList(input)
  if len(*result) == 0 {
    t.Fatal("No services returned")
  }
  for i, r := range(*result) {
    e := expected[i]
    if r.Name != e.Name || r.MountPoint != e.MountPoint {
      t.Errorf("Expected:\n %v \nBut got:\n %v", expected, result)
    }
  }
}

func TestAddNodesToService(t *testing.T) {
  service := &Service{Name: "solr", MountPoint: "/solr"}

  expected := &Service{Name: "solr",
    MountPoint: "/solr",
    Port: 8983,
    Nodes: []*api.Node{
      &api.Node{Node: "solr1", Address: "solr1.example.com"},
      &api.Node{Node: "solr2", Address: "solr2.example.com"},
    },
  }

  consulService := &api.AgentService{
    ID: "solr",
    Service: "solr",
    Port: 8983,
  }
  consulInput := []*api.ServiceEntry{
    &api.ServiceEntry{Node: &api.Node{Address: "solr1.example.com", Node: "solr1"},
      Service: consulService},
    &api.ServiceEntry{Node: &api.Node{Address: "solr2.example.com", Node: "solr2"},
      Service: consulService},
  }

  result := *consul.AddNodesToService(service, consulInput)

  if result.Port != expected.Port {
    t.Errorf("expected port: %d but got port: %d", expected.Port, result.Port)
  }

  if len(result.Nodes) == 0 {
    t.Fatal("No nodes returned")
  }

  for i, e := range(expected.Nodes) {
    r := result.Nodes[i]
    if r == nil || r.Address != e.Address || r.Node != e.Node {
      t.Errorf("Expected:\n %+v \nBut got:\n %+v", e, r)
    }
  }
}
