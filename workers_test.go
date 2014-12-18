package main

import (
	"net/url"
	"testing"
)

var service Service

func init() {
	service = Service{Name: "solr",
	MountPoint: "/solr",
	Nodes: []Node{
		Node{Name: "solr1", Address: "solr1.example.com", Port: 8983},
		Node{Name: "solr2", Address: "solr2.example.com", Port: 8984},
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

	if server.Host != "solr2.example.com:8984" {
		t.Errorf("Expected server host to be 'solr2.example.com:8984' but got '%+v'", server.Host)
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

	newService := Service{Name: "backend",
	MountPoint: "/backend",
	Nodes: []Node{
		Node{Name: "backend1", Address: "backend1.example.com", Port: 1234},
		Node{Name: "backend2", Address: "backend2.example.com", Port: 4567},
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
