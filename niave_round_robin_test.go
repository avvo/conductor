package main

import (
	api "github.com/armon/consul-api"
	"net/url"
	"testing"
)

var nrr func() url.URL

func init() {
	s := Service{Name: "solr",
		MountPoint: "/solr",
		Port:       8983,
		Nodes: []*api.Node{
			&api.Node{Node: "solr1", Address: "solr1.example.com"},
			&api.Node{Node: "solr2", Address: "solr2.example.com"},
		},
	}
	nrr = NewNiaveRoundRobin(s)
}

func TestNiaveRoundRobinNext(t *testing.T) {
	r := nrr()
	if r.Scheme != "http" {
		t.Errorf("Expected scheme to be 'http' but got '%s'", r.Scheme)
	}

	if r.Host != "solr1.example.com:8983" {
		t.Fatalf("Expected first call to NRR to return host of 'solr1.example.com:8983' but got '%v'", r.Host)
	}

	r = nrr()

	if r.Scheme != "http" {
		t.Errorf("Expected scheme to be 'http' but got '%s'", r.Scheme)
	}

	if r.Host != "solr2.example.com:8983" {
		t.Fatalf("Expected second call to NRR to return host of 'solr2.example.com:8983' but got '%v'", r.Host)
	}

	r = nrr()

	if r.Scheme != "http" {
		t.Errorf("Expected scheme to be 'http' but got '%s'", r.Scheme)
	}

	if r.Host != "solr1.example.com:8983" {
		t.Fatalf("Expected third call to NRR to return host of 'solr1.example.com:8983' but got '%v'", r.Host)
	}
}
