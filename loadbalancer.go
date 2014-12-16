package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http/httputil"
	"net/url"
)

type LoadBalancer struct {
	// This is for storing the function that we can use to rebuild the loadbalancer
	// functions later when we reload config.
	BuilderFunction func(Service) func() url.URL
	// Services holds the mount point to loadbalancer function mapping
	Services ServiceList
	// Keys are mount points, values are the loadbalancing function for that service
	MountPointToReverseProxyMap map[string]*httputil.ReverseProxy

	// List of all the workers
	Workers map[string]*LoadBalancerWorker
}

func NewLoadBalancer(services ServiceList, builder func(Service) func() url.URL) *LoadBalancer {
	lb.Services = services
	lb := &LoadBalancer{BuilderFunction: builder}
	lb.Reload(services)
	return lb
}

func (lb *LoadBalancer) Reload(services ServiceList) {
	lb.MountPointToReverseProxyMap = lb.GenerateReverseProxyMap()
}

// Builds the HTTP Proxy map like so: {"/solr": http.HandlerFunc()}
func (lb *LoadBalancer) GenerateReverseProxyMap() map[string]*httputil.ReverseProxy {
	m := make(map[string]*httputil.ReverseProxy)
	for mountPoint, w := range lb.Workers {
		m[mountPoint] = NewReverseProxyWithLoadBalancer(mountPoint, w.RequestChan)
	}
	return m
}

func (lb *LoadBalancer) StartWorkers() {
	// Create the channels and start the workers
	for _, s := range lb.Services {
		log.WithFields(log.Fields{"mount_point": s.MountPoint,
			"name": s.Name}).Debug("Starting Loadbalancer Worker")
		w := NewLoadBalancerWorker(lb.BuilderFunction)
		lb.Workers[s.MountPoint] = w
		go w.Work(*s)
	}
}
