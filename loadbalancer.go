package conductor

import (
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
  if(lb.Workers[s.MountPoint] != nil) {
    // already added this worker
    return
  }
  w := lb.StartWorker(s)
  lb.StartHealthWorker(s, w)
  lb.AddHttpHandler(s, w)
}

func (lb *LoadBalancer) StartWorker(s *Service) *LoadBalancerWorker{
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
  lb.MountPointToReverseProxyMap[s.MountPoint] = NewReverseProxyWithLoadBalancer(s.MountPoint, w.RequestChan)

  // Launch health workers
  for mp, rp := range lb.MountPointToReverseProxyMap {
    log.WithFields(log.Fields{"mount_point": mp}).Debug("Adding mountpoint handler function")
    http.HandleFunc(fmt.Sprintf("%s/", mp), rp.ServeHTTP)
  }
}

func (lb *LoadBalancer) StartHttpServer(port int) error {
  // Start listening
  return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (lb *LoadBalancer) Cleanup() {

}
