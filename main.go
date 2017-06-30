package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"os"
)

const Version = "0.2.5"
const CodeName = "The Canadian Dream"

type Config struct {
	ConsulHost       string
	ConsulDataCenter string
	LoadBalancer     string
	LogLevel         string
	LogFormat        string
	KVPrefix         string
	Port             int
	Version          bool
}

// Initialize the Configuration struct
var config Config

// Parse commandline and setup logging
func init() {

	flag.StringVar(&config.ConsulHost, "consul", "localhost:8500",
		"The Consul Host to connect to")
	flag.StringVar(&config.ConsulDataCenter, "datacenter", "dc1",
		"The Consul Datacenter use")
	flag.StringVar(&config.LoadBalancer, "loadbalancer",
		"niave_round_robin",
		"The loadbalancer algorithm")
	flag.StringVar(&config.LogFormat, "log-format", "lsmet",
		"Format logs in this format (either 'json' or 'lsmet')")
	flag.StringVar(&config.LogLevel, "log-level", "info",
		"Log level to use (debug, info, warn, error, fatal, or panic)")
	flag.StringVar(&config.KVPrefix, "kv-prefix", "conductor/services",
		"The Key Value prefix in consul to search for services under")
	flag.IntVar(&config.Port, "port", 8888, "Listen on this port")
	flag.BoolVar(&config.Version, "version", false, "Print version and exit")

	flag.Parse()

	// Load Environment Variables to override flags
	override_with_env_var(&config.ConsulHost, "CONSUL_HOST")
	override_with_env_var(&config.ConsulDataCenter, "CONSUL_DATACENTER")
	override_with_env_var(&config.KVPrefix, "CONSUL_KV_PREFIX")
	override_with_env_var(&config.LoadBalancer, "LOADBALANCER")
	override_with_env_var(&config.LogFormat, "LOG_FORMAT")
	override_with_env_var(&config.LogLevel, "LOG_LEVEL")

	logLevelMap := map[string]log.Level{
		"debug": log.DebugLevel,
		"info":  log.InfoLevel,
		"warn":  log.WarnLevel,
		"error": log.ErrorLevel,
		"fatal": log.FatalLevel,
		"panic": log.PanicLevel,
	}

	log.SetLevel(logLevelMap[config.LogLevel])

	if config.LogFormat == "json" {
		log.SetFormatter(new(log.JSONFormatter))
	}
}

func main() {
	if config.Version {
		fmt.Printf("Conductor %s, '%s'\n", Version, CodeName)
		os.Exit(0)
	}

	log.WithFields(log.Fields{"version": Version,
		"code_name": CodeName}).Info("Starting Conductor")
	log.WithFields(log.Fields{"consul": config.ConsulHost,
		"data_center": config.ConsulDataCenter}).Debug("Connecting to consul")

	// Create the consul connection object
	consul, err := NewConsul(config.ConsulHost, config.ConsulDataCenter, config.KVPrefix)

	// Failed to connect
	if err != nil {
		log.WithFields(log.Fields{"consul": config.ConsulHost,
			"data_center": config.ConsulDataCenter,
			"error":       err, "action": "connect"}).Error("Could not connect to consul!")
		os.Exit(1)
	}

	log.WithFields(log.Fields{"consul": config.ConsulHost,
		"data_center": config.ConsulDataCenter}).Debug("Connected to consul successfully.")

	log.WithFields(log.Fields{"consul": config.ConsulHost,
		"data_center": config.ConsulDataCenter,
		"kv_prefix":   config.KVPrefix}).Debug("Pulling load balanceable service list")

	// Pull Servers from Consul
	serviceList, err := consul.GetListOfServices()
	if err != nil {
		log.WithFields(log.Fields{"consul": config.ConsulHost,
			"data_center": config.ConsulDataCenter,
			"error":       err, "action": "GetListOfServices"}).Error("Could not connect to consul!")
		os.Exit(1)
	}

	log.WithFields(log.Fields{"services": len(*serviceList),
		"data_center": config.ConsulDataCenter,
		"kv_prefix":   config.KVPrefix}).Debug("Retrieved services")

	// We don't have any services in Consul to proxy
	if len(*serviceList) < 1 {
		log.WithFields(log.Fields{"consul": config.ConsulHost,
			"data_center": config.ConsulDataCenter,
			"kv_prefix":   config.KVPrefix}).Error("Found no services to proxy!")
		os.Exit(1)
	}

	log.WithFields(log.Fields{"services": len(*serviceList),
		"data_center": config.ConsulDataCenter,
		"kv_prefix":   config.KVPrefix}).Debug("Pulling healthy nodes for services")

	// Pull the healthy nodes
	serviceList, err = consul.GetAllHealthyNodes(serviceList)
	if err != nil {
		log.WithFields(log.Fields{"consul": config.ConsulHost,
			"data_center": config.ConsulDataCenter,
			"error":       err,
			"action":      "GetAllHealthyNodes"}).Error("Could not connect to consul!")
		os.Exit(1)
	}

	log.WithFields(log.Fields{"services": len(*serviceList),
		"balancing_algorithm": config.LoadBalancer}).Debug("Setting up loadbalancer")

	// Laucnch loadbalancers
	lb := NewLoadBalancer(serviceList, NewNiaveRoundRobin)
	lb.StartWorkers()
	lb.GenerateReverseProxyMap()

	healthWorkers := make(map[string]*ConsulHealthWorker)

	// Launch health workers
	for _, service := range lb.Services {
		log.WithFields(log.Fields{"service": service.Name,
			"mount_point": service.MountPoint}).Debug("Starting consul health worker")
		lbw := lb.Workers[service.MountPoint]
		worker := NewConsulHealthWorker(consul, *service, lbw)
		healthWorkers[service.MountPoint] = worker
		go worker.Work()
	}

	for mp, rp := range lb.MountPointToReverseProxyMap {
		log.WithFields(log.Fields{"mount_point": mp}).Debug("Adding mountpoint handler function")
		http.HandleFunc(fmt.Sprintf("%s/", mp), rp.ServeHTTP)
	}

	http.HandleFunc("/", noMatchingMountPointHandler)
	http.HandleFunc("/_ping", pingHandler)

	log.WithFields(log.Fields{
		"port":    config.Port,
		"address": "0.0.0.0",
		"status":  "running",
	}).Info("Up and running")

	// Start listening
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatal(err)
	}
	exit(lb, healthWorkers)
}

func exit(lb *LoadBalancer, healthWorkers map[string]*ConsulHealthWorker) {
	for mp, w := range lb.Workers {
		log.WithFields(log.Fields{"mount_point": mp}).Debug("Telling loadbalancer worker to quit")
		w.ControlChan <- true
	}

	for mp, w := range healthWorkers {
		log.WithFields(log.Fields{"mount_point": mp}).Debug("Telling consul health worker to quit")
		w.ControlChan <- true
	}
}

func override_with_env_var(config_var *string, env string) {
	value := os.Getenv(env)
	if value != "" {
		*config_var = value
	}
}
