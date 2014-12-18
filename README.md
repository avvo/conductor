Conductor
=========
The dynamically reconfigurable Consul reverse HTTP proxy.

Summary
-------
Conductor creates a layer 7 reverse HTTP proxy that will map arbitrary mount
points to underlying Consul services. It relies on Consul for Health checks and
configuration information.

What it does not do
-------------------
* It will not proxy non HTTP services
* Check for down endpoints
* Your laundry

Status
======
* Need to add blocking queries to the Consul side so that we can dynamically update
* Make it actually proxy stuff

How it works
============
On boot Conductor does this:
* Connects to consul and pulls all keys underneath the KV Prefix (`--kv-prefix=XXX`)
* These key names are assumed to be Consul service names that you want proxied
* The values for these keys need to be mount points (URL prefixes if you prefer)
* Conductor then pulls all the healthy nodes from Consul and starts proxying
* Conductor fires up background processes that watch the consul KV Prefix and
the healthy nodes for each service
* If new services are added to the KV Prefix it will regenerate its config and
send new traffic to those services
* If nodes become unhealthy or new nodes are added, Conductor reconfigures itself

Load Testing
============

The load testing script will fire up a group of servers, register them in consul
and then run siege against conductor. Eventually I will add a few steps where
dynamic scaling of the application layer is done so that we can see how cleanly
conductor handles backend changes.

```
boot2docker start
brew install siege
docker built -t conductor .
./loadtest.sh
```
