package main

import (
  "flag"
  "fmt"
  log "github.com/Sirupsen/logrus"
  "os"
  conductor "../../"
  "time"
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
  flag.StringVar(&config.LoadBalancer, "loadbalancer", "naive_round_robin",
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

  if config.Version {
    fmt.Printf("Conductor %s, '%s'\n", Version, CodeName)
    os.Exit(0)
  }

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
  consul, err := conductor.NewConsul(config.ConsulHost, config.ConsulDataCenter, config.KVPrefix)
  if err != nil {
    log.WithFields(log.Fields{"consul": config.ConsulHost,
      "data_center": config.ConsulDataCenter,
      "error":       err, "action": "connect"}).Error("Could not connect to consul!")
    os.Exit(1)
  }

  lb := conductor.NewLoadBalancer(conductor.NewNaiveRoundRobin, consul)

  go monitorConsulServices(lb, consul)

  err = lb.StartHttpServer(config.Port)
  if err != nil {
    log.Fatal(err)
  }
  lb.Cleanup()
}

// recursively monitor consul and update the loadbalancer endpoints
func monitorConsulServices(lb *conductor.LoadBalancer, consul *conductor.Consul) {
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

  // try and add all services
  for _, service := range *serviceList {
    lb.AddService(service)
  }

  // remove services not in the serviceList
  for name, _ := range lb.Services {
    if !serviceList.HasServiceNamed(name) {
      fmt.Println("didn't find service: " + name + " - removing")
      lb.RemoveService(name)
    }
  }

  // sleep 1 minute
  fmt.Println("sleeping...")
  time.Sleep(time.Duration(1)*time.Minute)

  monitorConsulServices(lb, consul)
}
