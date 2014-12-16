package main

import (
	"net/url"
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
	rchan := make(chan *chan url.URL, 16)
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
