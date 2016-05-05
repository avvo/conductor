package conductor

import (
  "fmt"
  "github.com/hashicorp/consul/api"
  "testing"
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
  input := api.KVPairs{
    &api.KVPair{Key: "conductor-services/solr", Value: []byte("L3NvbHI=")},
    &api.KVPair{Key: "conductor-services/backend_service_v1", Value: []byte("L3NlcnZpY2UvdjE=")},
  }

  result := consul.MapKVPairsToServiceList(input)
  if len(*result) == 0 {
    t.Fatal("No services returned")
  }
  for i, r := range *result {
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
    Nodes: []Node{
      Node{Name: "solr1", Address: "solr1.example.com", Port: 8983},
      Node{Name: "solr2", Address: "solr2.example.com", Port: 8984},
    },
  }

  consulInput := []*api.ServiceEntry{
    &api.ServiceEntry{
      Node: &api.Node{Address: "solr1.example.com", Node: "solr1"},
      Service: &api.AgentService{
        ID:      "solr",
        Service: "solr",
        Port:    8983,
      },
    },
    &api.ServiceEntry{Node: &api.Node{Address: "solr2.example.com", Node: "solr2"},
      Service: &api.AgentService{
        ID:      "solr",
        Service: "solr",
        Port:    8984,
      },
    },
  }

  result := *consul.AddNodesToService(service, consulInput)

  if len(result.Nodes) == 0 {
    t.Fatal("No nodes returned")
  }

  for i, n := range result.Nodes {
    if n.Port != expected.Nodes[i].Port {
      t.Errorf("expected port: %d but got port: %d", expected.Nodes[i].Port, n.Port)
    }

    if n.Address != expected.Nodes[i].Address || n.Name != expected.Nodes[i].Name {
      t.Errorf("Expected:\n %+v \nBut got:\n %+v", n, expected.Nodes[i])
    }
  }
}
