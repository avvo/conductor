package main

import (
	api "github.com/armon/consul-api"
	"testing"
)

var services ServiceList

func init() {
	solr := &Service{Name: "solr",
		MountPoint: "/solr",
		Port:       8983,
		Nodes: []*api.Node{
			&api.Node{Node: "solr1", Address: "solr1.example.com"},
			&api.Node{Node: "solr2", Address: "solr2.example.com"},
		},
	}
	backend_v1 := &Service{Name: "backend_v1",
		MountPoint: "/backend/v1",
		Port:       8001,
		Nodes: []*api.Node{
			&api.Node{Node: "backend1", Address: "backend1.example.com"},
			&api.Node{Node: "backend2", Address: "backend2.example.com"},
		},
	}
	backend_v2 := &Service{Name: "backend_v2",
		MountPoint: "/backend/v2",
		Port:       8002,
		Nodes: []*api.Node{
			&api.Node{Node: "backend3", Address: "backend3.example.com"},
			&api.Node{Node: "backend4", Address: "backend4.example.com"},
		},
	}

	services = append(services, solr)
	services = append(services, backend_v1)
	services = append(services, backend_v2)
}

func TestGenerateMountPointMap(t *testing.T) {
	l := &LoadBalancer{BuilderFunction: NewNiaveRoundRobin}
	m := l.GenerateMountPointMap(&services)
	solr := m["/solr"]
	bv1 := m["/backend/v1"]
	bv2 := m["/backend/v2"]

	if solr == nil {
		t.Fatal("solr mount point has no value")
	}
	r := solr()
	if r.Host != "solr1.example.com:8983" {
		t.Errorf("Expected first call to solr mountpoint to be 'solr1.example.com:8983' but got: '%s'", r.Host)
	}

	if bv1 == nil {
		t.Fatal("bv1 mount point has no value")
	}
	r = bv1()
	if r.Host != "backend1.example.com:8001" {
		t.Errorf("Expected first call to bv1 mountpoint to be 'backendv1.example.com:8001' but got: '%s'", r.Host)
	}

	if bv2 == nil {
		t.Fatal("bv2 mount point has no value")
	}
	r = bv2()
	if r.Host != "backend3.example.com:8002" {
		t.Errorf("Expected first call to bv2 mountpoint to be 'backendv3.example.com:8002' but got: '%s'", r.Host)
	}
}

func TestLoadBalancerReload(t *testing.T) {
	l := LoadBalancer{BuilderFunction: NewNiaveRoundRobin}
	l.Reload(&services)

  bv1 := l.MountPointToLoadBalancerFuncMap["/backend/v1"]
  if bv1 == nil {
    t.Fatal("bv1 mount point has no value")
  }
  r := bv1()
  if r.Host != "backend1.example.com:8001" {
    t.Errorf("Expected first call to bv1 mountpoint to be 'backendv1.example.com:8001' but got: '%s'", r.Host)
  }
}

func TestNextServerForPath(t *testing.T) {
  lb := NewLoadBalancer(&services, NewNiaveRoundRobin)
  r, err := lb.NextServerForPath("/solr/admin/file/?contentType=text/xml;charset=utf-8&file=solrconfig.xml")

  if err != nil {
    t.Fatal("solr path should not have raised an error")
  }
  if r.Host != "solr1.example.com:8983" {
    t.Errorf("Expected first url for solr path to be 'solr1.example.com:8983' but got: '%s'", r.Host)
  }

  r, err = lb.NextServerForPath("/backend/v1/users")

  if err != nil {
    t.Fatal("backend v1 path should not have raised an error")
  }
  if r.Host != "backend1.example.com:8001" {
    t.Errorf("Expected first url for solr path to be 'backend1.example.com:8001' but got: '%s'", r.Host)
  }

  r, err = lb.NextServerForPath("/this/is/unroutable.json")

  if err == nil {
    t.Error("Non routed path should have raised an error")
  }
}
