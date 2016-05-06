package conductor

import (
	"net/url"
	"testing"
)

var nrr func() url.URL

func init() {
	s := Service{Name: "solr",
		MountPoint: "/solr",
		Nodes: []Node{
			Node{Name: "solr1", Address: "solr1.example.com", Port: 8983},
			Node{Name: "solr2", Address: "solr2.example.com", Port: 8984},
		},
	}
	nrr = NewNaiveRoundRobin(s)
}

func TestNaiveRoundRobinNext(t *testing.T) {
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

	if r.Host != "solr2.example.com:8984" {
		t.Fatalf("Expected second call to NRR to return host of 'solr2.example.com:8984' but got '%v'", r.Host)
	}

	r = nrr()

	if r.Scheme != "http" {
		t.Errorf("Expected scheme to be 'http' but got '%s'", r.Scheme)
	}

	if r.Host != "solr1.example.com:8983" {
		t.Fatalf("Expected third call to NRR to return host of 'solr1.example.com:8983' but got '%v'", r.Host)
	}
}
