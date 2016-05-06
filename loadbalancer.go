package conductor

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type LoadBalancer struct {
	// This is for storing the function that we can use to rebuild the loadbalancer
	// functions later when we reload config.
	BuilderFunction func(Service) func() url.URL

	// Keys are mount points, values are the loadbalancing function for that service
	MountPointToReverseProxyMap map[string]*httputil.ReverseProxy

	// Keys are service names, values are the service definition
	Services map[string]*Service

	// List of all the workers
	Workers map[string]*LoadBalancerWorker

	HealthWorkers map[string]*ConsulHealthWorker

	Mux *Remux

	ConsulConnection *Consul
}

func NewLoadBalancer(builder func(Service) func() url.URL, c *Consul) *LoadBalancer {
	lb := &LoadBalancer{BuilderFunction: builder}

	// Create the channels and start the workers
	lb.ConsulConnection = c
	lb.Services = make(map[string]*Service)
	lb.Workers = make(map[string]*LoadBalancerWorker)
	lb.MountPointToReverseProxyMap = make(map[string]*httputil.ReverseProxy)
	lb.HealthWorkers = make(map[string]*ConsulHealthWorker)
	lb.Mux = NewRemux()

	return lb
}

func (lb *LoadBalancer) AddService(s *Service) {
	if lb.Services[s.Name] != nil {
		// already added this worker
		return
	}
	log.WithFields(log.Fields{"service": s.Name, "mount_point": s.MountPoint}).Debug("Adding service")

	lb.Services[s.Name] = s

	w := lb.StartWorker(s)
	lb.StartHealthWorker(s, w)
	lb.AddHttpHandler(s, w)
}

func (lb *LoadBalancer) StartWorker(s *Service) *LoadBalancerWorker {
	log.WithFields(log.Fields{"mount_point": s.MountPoint,
		"service": s.Name}).Debug("Starting Loadbalancer Worker")
	w := NewLoadBalancerWorker(lb.BuilderFunction)
	lb.Workers[s.Name] = w
	go w.Work(*s)
	return w
}

func (lb *LoadBalancer) StartHealthWorker(s *Service, w *LoadBalancerWorker) {
	log.WithFields(log.Fields{"service": s.Name,
		"mount_point": s.MountPoint}).Debug("Starting consul health worker")
	worker := NewConsulHealthWorker(lb.ConsulConnection, *s, w)
	lb.HealthWorkers[s.Name] = worker
	go worker.Work()
}

func (lb *LoadBalancer) AddHttpHandler(s *Service, w *LoadBalancerWorker) {
	// Builds the HTTP Proxy map like so: {"/solr": http.HandlerFunc()}
	rp := NewReverseProxyWithLoadBalancer(s.MountPoint, w.RequestChan)
	lb.MountPointToReverseProxyMap[s.MountPoint] = rp

	log.WithFields(log.Fields{"mount_point": s.MountPoint}).Debug("Adding mountpoint handler function")
	lb.Mux.HandleFunc(fmt.Sprintf("%s/", s.MountPoint), rp.ServeHTTP)
}

func (lb *LoadBalancer) RemoveService(name string) {
	if lb.Services[name] != nil {
		s := lb.Services[name]
		log.WithFields(log.Fields{"service": s.Name, "mount_point": s.MountPoint}).Info("Removing service")
		delete(lb.Services, name)

		// clean up worker
		w := lb.Workers[name]
		log.WithFields(log.Fields{"mount_point": s.MountPoint}).Debug("Telling loadbalancer worker to quit")
		w.ControlChan <- true
		delete(lb.Workers, name)

		// clean up health worker
		hw := lb.HealthWorkers[name]
		log.WithFields(log.Fields{"mount_point": s.MountPoint}).Debug("Telling consul health worker to quit")
		hw.ControlChan <- true
		delete(lb.HealthWorkers, name)

		// deregister the http listener
		lb.Mux.Deregister(s.MountPoint)
		delete(lb.MountPointToReverseProxyMap, s.MountPoint)
	} else {
		log.WithFields(log.Fields{"service": name}).Warn("Couldn't find registered service to remove!")
	}
}

func (lb *LoadBalancer) ListEndpointsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		json, err := json.Marshal(lb.Services)
		if err != nil {

		}
		fmt.Fprintf(w, `{"endpoints":[%s]}`, string(json))
	}
}

func (lb *LoadBalancer) StartHttpServer(port int) error {
	// Start listening
	lb.Mux.HandleFunc("/_endpoints", lb.ListEndpointsHandler())
	lb.Mux.HandleFunc("/", noMatchingMountPointHandler)
	lb.Mux.HandleFunc("/_ping", pingHandler)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), lb.Mux)
}
