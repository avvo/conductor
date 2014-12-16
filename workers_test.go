package main

import (
	api "github.com/armon/consul-api"
  "net/url"
	"testing"
)

var service Service

func init() {
  service = Service{Name: "solr",
  MountPoint: "/solr",
  Port:       8983,
  Nodes: []*api.Node{
    &api.Node{Node: "solr1", Address: "solr1.example.com"},
    &api.Node{Node: "solr2", Address: "solr2.example.com"},
    },
  }
}

func TestRequestFromWorker(t *testing.T) {
  w := NewLoadBalancerWorker(NewNiaveRoundRobin)
  w.Work(service)
  response := make(chan url.URL, 1)
  // send our channel to the worker
  w.RequestChan <- &response
  // Get the server URL as the response back on the channel
  server := <-response

  if server.Host != "solr1.example.com" {
    t.Errorf("Expected server host to be 'solr1.example.com' but got '%+v'", server.Host)
  }
  w.ControlChan <- true
}
