package main

import (
	"net/http/httputil"
	"net/url"
	"strings"
)

type LoadBalancer struct {
	// This is for storing the function that we can use to rebuild the loadbalancer
	// functions later when we reload config.
	BuilderFunction func(Service) func() url.URL
	// Services holds the mount point to loadbalancer function mapping
	Services []*Service
	// Keys are mount points, values are the loadbalancing function for that service
	MountPointToLoadBalancerFuncMap map[string](func() url.URL)
	MountPointToReverseProxyMap     map[string]*httputil.ReverseProxy

	// The channels we will use to talk to the workers
	WorkerRequestChans map[string]chan *chan url.URL
	// The channel we use to tell workers to exit
	WorkerControlChans map[string]chan bool
	// The channel we use to send updated server lists
	WorkerUpdateChans map[string]chan Service
}

func NewLoadBalancer(services *ServiceList, builder func(Service) func() url.URL) *LoadBalancer {
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
		m[s.MountPoint] = lb.BuilderFunction(*s)
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
	for mountPoint, rchan := range lb.WorkerRequestChans {
		m[mountPoint] = NewReverseProxyWithLoadBalancer(mountPoint, rchan)
	}
	return m
}

func (lb *LoadBalancer) StartWorkers() {
	// Create the maps to find these channels later
	updateChannels := make(map[string]chan Service)
	controlChannels := make(map[string]chan bool)
	requestChannels := make(map[string]chan *chan url.URL)

	// Create the channels and start the workers
	for _, s := range services {
		w := NewLoadBalancerWorker(lb.BuilderFunction)
		updateChannels[s.MountPoint] = w.UpdateChan
		controlChannels[s.MountPoint] = w.ControlChan
		requestChannels[s.MountPoint] = w.RequestChan
		go w.Work(*s)
	}

	// Save the maps to this object
	lb.WorkerUpdateChans = updateChannels
	lb.WorkerControlChans = controlChannels
	lb.WorkerRequestChans = requestChannels
}
