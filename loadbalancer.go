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

	// List of all the workers
	Workers map[string]*LoadBalancerWorker

	HealthWorkers map[string]*ConsulHealthWorker

	ConsulConnection *Consul
}

func NewLoadBalancer(builder func(Service) func() url.URL, c *Consul) *LoadBalancer {
	lb := &LoadBalancer{BuilderFunction: builder}

	// Create the channels and start the workers
	lb.ConsulConnection = c
	lb.Workers = make(map[string]*LoadBalancerWorker)
	lb.MountPointToReverseProxyMap = make(map[string]*httputil.ReverseProxy)
	lb.HealthWorkers = make(map[string]*ConsulHealthWorker)

	return lb
}

func (lb *LoadBalancer) AddService(s *Service) {
	fmt.Println("adding service: " + s.Name)
	if lb.Workers[s.MountPoint] != nil {
		// already added this worker
		fmt.Println("already added: " + s.Name)
		return
	}
	w := lb.StartWorker(s)
	lb.StartHealthWorker(s, w)
	lb.AddHttpHandler(s, w)
}

func (lb *LoadBalancer) StartWorker(s *Service) *LoadBalancerWorker {
	log.WithFields(log.Fields{"mount_point": s.MountPoint,
		"service": s.Name}).Debug("Starting Loadbalancer Worker")
	w := NewLoadBalancerWorker(lb.BuilderFunction)
	lb.Workers[s.MountPoint] = w
	go w.Work(*s)
	return w
}

func (lb *LoadBalancer) StartHealthWorker(s *Service, w *LoadBalancerWorker) {
	log.WithFields(log.Fields{"service": s.Name,
		"mount_point": s.MountPoint}).Debug("Starting consul health worker")
	worker := NewConsulHealthWorker(lb.ConsulConnection, *s, w)
	lb.HealthWorkers[s.MountPoint] = worker
	go worker.Work()
}

func (lb *LoadBalancer) AddHttpHandler(s *Service, w *LoadBalancerWorker) {
	// Builds the HTTP Proxy map like so: {"/solr": http.HandlerFunc()}
	rp := NewReverseProxyWithLoadBalancer(s.MountPoint, w.RequestChan)
	lb.MountPointToReverseProxyMap[s.MountPoint] = rp

	log.WithFields(log.Fields{"mount_point": s.MountPoint}).Debug("Adding mountpoint handler function")
	http.HandleFunc(fmt.Sprintf("%s/", s.MountPoint), rp.ServeHTTP)
}

func (lb *LoadBalancer) ListEndpointsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		endpoints := make(map[string]string)
		for mount, _ := range lb.MountPointToReverseProxyMap {
			endpoints[mount] = mount
		}
		json, err := json.Marshal(endpoints)
		if err != nil {

		}
		fmt.Fprintf(w, `{"endpoints":[%s]}`, string(json))
	}
}

func (lb *LoadBalancer) StartHttpServer(port int) error {
	// Start listening
	http.HandleFunc("/_endpoints", lb.ListEndpointsHandler())
	http.HandleFunc("/", noMatchingMountPointHandler)
	http.HandleFunc("/_ping", pingHandler)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (lb *LoadBalancer) Cleanup() {

}
