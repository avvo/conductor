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
	go w.Work(service)
	response := make(chan url.URL, 1)
	// send our channel to the worker
	w.RequestChan <- &response
	// Get the server URL as the response back on the channel
	server := <-response

	if server.Host != "solr1.example.com:8983" {
		t.Errorf("Expected server host to be 'solr1.example.com:8983' but got '%+v'", server.Host)
	}

	w.RequestChan <- &response
	// Get the server URL as the response back on the channel
	server = <-response

	if server.Host != "solr2.example.com:8983" {
		t.Errorf("Expected server host to be 'solr2.example.com:8983' but got '%+v'", server.Host)
	}

	w.ControlChan <- true
}

func TestReconfiguringWorker(t *testing.T) {
	w := NewLoadBalancerWorker(NewNiaveRoundRobin)
	go w.Work(service)
	response := make(chan url.URL, 1)
	// send our channel to the worker
	w.RequestChan <- &response
	// Get the server URL as the response back on the channel
	server := <-response

	// Verify the first result is still the original service
	if server.Host != "solr1.example.com:8983" {
		t.Errorf("Expected server host to be 'solr1.example.com:8983' but got '%+v'", server.Host)
	}

	newService := Service{Name: "solr",
		MountPoint: "/solr",
		Port:       1234,
		Nodes: []*api.Node{
			&api.Node{Node: "backend1", Address: "backend1.example.com"},
			&api.Node{Node: "backend2", Address: "backend2.example.com"},
		},
	}

	w.UpdateChan <- newService
	w.RequestChan <- &response
	// Get the server URL as the response back on the channel
	server = <-response

	if server.Host != "backend1.example.com:1234" {
		t.Errorf("Expected server host to be 'backend1.example.com:1234' but got '%+v'", server.Host)
	}

	w.ControlChan <- true
}
