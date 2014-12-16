package main

import (
	"net/http/httputil"
	"net/url"
	"strings"
)

type LoadBalancer struct {
	// This is for storing the function that we can use to rebuild the loadbalancer
	// functions later when we reload config.
	BuilderFunction func(*Service) func() url.URL
	// Services holds the mount point to loadbalancer function mapping
	Services []*Service
	// Keys are mount points, values are the loadbalancing function for that service
	MountPointToLoadBalancerFuncMap map[string](func() url.URL)
	MountPointToReverseProxyMap     map[string]*httputil.ReverseProxy

	// The channels we will use to talk to the workers
	WorkerRequestChans map[string]chan *chan string
	// The channel we use to tell workers to exit
	WorkerControlChans map[string]chan bool
}

func NewLoadBalancer(services *ServiceList, builder func(*Service) func() url.URL) *LoadBalancer {
	lb := &LoadBalancer{BuilderFunction: builder}
	lb.Reload(services)
	return lb
}

func (lb *LoadBalancer) Reload(services *ServiceList) {
	lb.MountPointToLoadBalancerFuncMap = lb.GenerateMountPointMap(services)
	lb.MountPointToReverseProxyMap = lb.GenerateReverseProxyMap()
}

// Takes the list of services and creates a map that looks something like this:
// {"/solr": func() string}
func (lb *LoadBalancer) GenerateMountPointMap(services *ServiceList) map[string](func() url.URL) {
	m := make(map[string](func() url.URL))
	for _, s := range *services {
		m[s.MountPoint] = lb.BuilderFunction(s)
	}
	return m
}

// Given a path ("/solr/search.jsp"), it will return the next server according
// to the loadbalancing algorithm
func (lb *LoadBalancer) NextServerForPath(path string) (url.URL, error) {
	var server url.URL
	for mountpoint, lbfunc := range lb.MountPointToLoadBalancerFuncMap {
		if strings.HasPrefix(path, mountpoint) {
			server = lbfunc()
			if server.Host == "" {
				return server, NewNoHealthyNodesError(mountpoint, path)
			}
			return server, nil
		}
	}
	return server, NewNoMatchingMountPointError(path)
}

// Builds the HTTP Proxy map like so: {"/solr": http.HandlerFunc()}
func (lb *LoadBalancer) GenerateReverseProxyMap() map[string]*httputil.ReverseProxy {
	m := make(map[string]*httputil.ReverseProxy)
	for mountPoint, lbfunc := range lb.MountPointToLoadBalancerFuncMap {
		m[mountPoint] = NewReverseProxyWithLoadBalancer(mountPoint, lbfunc)
	}
	return m
}

func (lb *LoadBalancer) Work(servers chan []string, request chan *chan string, control chan bool) {
	servers := make([]string, 1)
	next := newLBFunc(servers)
	for {
		select {
		case s := <-newServers:
			servers = s
			next = newLBFunc(servers)
		case outputChan := <-requests:
			*outputChan <- next()
		case _ = <-control:
			return
		}
	}
}
