package main

import (
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
	InputChan chan []*api.ServiceEntry
}

func NewConsulHealthWorker(c *Consul, service Service, lbworker *LoadBalancerWorker) *ConsulHealthWorker {
	return &ConsulHealthWorker{
		service:            service,
		consul:             c,
		loadbalancerWorker: lbworker,
		ControlChan:        make(chan bool),
		InputChan:          make(chan []*api.ServiceEntry, 1),
		queryOptions:       &api.QueryOptions{WaitTime: time.Duration(30) * time.Second, RequireConsistent: true},
	}
}

func (w *ConsulHealthWorker) Work() {
	for {
		select {
		case result := <-w.InputChan:
			if result[0] != nil {
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
	services, queryMeta, err := w.consul.Client.Health().Service(w.service.Name, "", true, w.queryOptions)
	if err != nil {
		w.InputChan <- []*api.ServiceEntry{}
		return
	}
	if w.lastIndex == 0 {
		w.InputChan <- services
		return
	}

	if queryMeta.LastIndex > w.lastIndex {
		w.lastIndex = queryMeta.LastIndex
		w.InputChan <- services
	} else {
		w.InputChan <- []*api.ServiceEntry{}
	}
	return
}
