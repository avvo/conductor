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
  for i, r := range(*result) {
    e := expected[i]
    if r.Name != e.Name || r.MountPoint != e.MountPoint {
      t.Error(fmt.Sprintf("Expected:\n %v \nBut got:\n %v", expected, result))
    }
  }
}
