package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"os"
)

const Version = "0.0.1"
const CodeName = "Sleeping Python"

type Config struct {
	ConsulHost       string
	ConsulDataCenter string
	LoadBalancer     string
	LogLevel         string
	LogFormat        string
	KVPrefix         string
	Port             int
}

// Initialize the Configuration struct
var config Config
var lb *LoadBalancer

// Parse commandline and setup logging
func init() {
	flag.StringVar(&config.ConsulHost, "consul", "localhost:8500",
		"The Consul Host to connect to")
	flag.StringVar(&config.ConsulDataCenter, "datacenter", "dc1",
		"The Consul Datacenter use")
	flag.StringVar(&config.LoadBalancer, "loadbalancer", "niave_round_robin",
		"The loadbalancer algorithm")
	flag.StringVar(&config.LogFormat, "log-format", "lsmet",
		"Format logs in this format (either 'json' or 'lsmet')")
	flag.StringVar(&config.LogLevel, "log-level", "info",
		"Log level to use (debug, info, warn, error, fatal, or panic)")
	flag.StringVar(&config.KVPrefix, "kv-prefix", "conductor-services",
		"The Key Value prefix in consul to search for services under")
	flag.IntVar(&config.Port, "port", 8888, "Listen on this port")

	flag.Parse()

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
	_, err = consul.GetAllHealthyNodes(serviceList)
	if err != nil {
		log.WithFields(log.Fields{"consul": config.ConsulHost,
			"data_center": config.ConsulDataCenter,
			"error":       err, "action": "GetAllHealthyNodes"}).Error("Could not connect to consul!")
		os.Exit(1)
	}

	log.WithFields(log.Fields{"services": len(*serviceList),
		"balancing_algorithm": config.LoadBalancer}).Debug("Setting up loadbalancer")

	lb := NewLoadBalancer(serviceList, NewNiaveRoundRobin)

	for mountPoint, rp := range lb.MountPointToReverseProxyMap {
		log.WithFields(log.Fields{"mount_point": mountPoint}).Debug("Adding mountpoint")
		http.HandleFunc(fmt.Sprintf("%s/", mountPoint), rp.ServeHTTP)
	}

	http.HandleFunc("/", noMatchingMountPointHandler)

	log.WithFields(log.Fields{"port": config.Port,
		"address": "0.0.0.0",
		"status":  "running",
	}).Info("Up and running")

	// Start listening
	// TODO: Figure out how to reconfigure dynamically with no downtime
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
