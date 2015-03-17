package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
	"net/url"
	"time"
)

type LoadBalancerWorker struct {
	ControlChan chan bool
	UpdateChan  chan Service
	RequestChan chan *chan url.URL
	BuilderFunc func(Service) func() url.URL
}

func NewLoadBalancerWorker(builderFunc func(Service) func() url.URL) *LoadBalancerWorker {
	svchan := make(chan Service)
	ctrlchan := make(chan bool)
	rchan := make(chan *chan url.URL, 64)
	return &LoadBalancerWorker{
		ControlChan: ctrlchan,
		UpdateChan:  svchan,
		RequestChan: rchan,
		BuilderFunc: builderFunc,
	}
}

// This is the core loadbalancer worker function. It just loops waiting for a
// HandlerFunc to send in a new request which is a channel to send the response
// back to. This will start with an initial service and update itself when given
// a new service via the services chan.
// This is necessary to allow for safe concurrent access to the server list.
func (w *LoadBalancerWorker) Work(initialService Service) {
	next := w.BuilderFunc(initialService)
	for {
		select {
		case s := <-w.UpdateChan:
			next = w.BuilderFunc(s)
		case outputChan := <-w.RequestChan:
			*outputChan <- next()
		case _ = <-w.ControlChan:
			return
		}
	}
}

type ConsulHealthWorker struct {
	waitTime           time.Duration
	service            Service
	queryOptions       *api.QueryOptions
	lastIndex          uint64
	ControlChan        chan bool
	consul             *Consul
	loadbalancerWorker *LoadBalancerWorker
	InputChan          chan []*api.ServiceEntry
}

func NewConsulHealthWorker(c *Consul, service Service, lbworker *LoadBalancerWorker) *ConsulHealthWorker {
	return &ConsulHealthWorker{
		service:            service,
		consul:             c,
		loadbalancerWorker: lbworker,
		ControlChan:        make(chan bool, 1),
		InputChan:          make(chan []*api.ServiceEntry, 1),
		queryOptions:       &api.QueryOptions{WaitTime: time.Duration(30) * time.Second, RequireConsistent: true},
	}
}

func (w *ConsulHealthWorker) Work() {
	go w.BlockUntilConsulUpdate()
	for {
		select {
		case result := <-w.InputChan:
			if len(result) != 0 {
				w.consul.AddNodesToService(&w.service, result)
				w.loadbalancerWorker.UpdateChan <- w.service
			}
			go w.BlockUntilConsulUpdate()
		case _ = <-w.ControlChan:
			return
		}
	}
}

func (w *ConsulHealthWorker) BlockUntilConsulUpdate() {
	log.WithFields(log.Fields{
		"mount_point":  w.service.MountPoint,
		"service_name": w.service.Name,
		"worker_type":  "consul_health"}).Debug("Getting service health from consul")

	services, queryMeta, err := w.consul.Client.Health().Service(w.service.Name, "", true, w.queryOptions)
	if err != nil {
		log.WithFields(log.Fields{
			"mount_point":  w.service.MountPoint,
			"service_name": w.service.Name,
			"error":        err,
			"last_index":   w.lastIndex,
			"worker_type":  "consul_health"}).Error("Error getting service health from consul")
		w.InputChan <- []*api.ServiceEntry{}
		return
	}

	if w.lastIndex == 0 {
		log.WithFields(log.Fields{
			"mount_point":  w.service.MountPoint,
			"service_name": w.service.Name,
			"last_index":   w.lastIndex,
			"new_index":    queryMeta.LastIndex,
			"worker_type":  "consul_health"}).Debug("Last index is zero, sending full service list")
		w.InputChan <- services
		w.lastIndex = queryMeta.LastIndex
		w.queryOptions.WaitIndex = queryMeta.LastIndex
		return
	}

	if queryMeta.LastIndex > w.lastIndex {
		log.WithFields(log.Fields{
			"mount_point":  w.service.MountPoint,
			"service_name": w.service.Name,
			"last_index":   w.lastIndex,
			"new_index":    queryMeta.LastIndex,
			"worker_type":  "consul_health"}).Debug("New index is larger than last index, sending full service list")
		w.lastIndex = queryMeta.LastIndex
		w.queryOptions.WaitIndex = queryMeta.LastIndex
		w.InputChan <- services
	} else {
		log.WithFields(log.Fields{
			"mount_point":  w.service.MountPoint,
			"service_name": w.service.Name,
			"last_index":   w.lastIndex,
			"new_index":    queryMeta.LastIndex,
			"worker_type":  "consul_health"}).Debug("New index is smaller than previous, so sending empty service list")
		w.InputChan <- []*api.ServiceEntry{}
	}
	return
}
