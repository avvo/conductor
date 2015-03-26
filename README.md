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
* It will not proxy non-HTTP services
* Check for down endpoints (use consul health checks for this)
* It does not detect new services added to the KV prefix without a restart
* Your laundry

Status
======
* Everything initially planned currently works.

How it works
============
On boot Conductor does this:
* Connects to consul and pulls all keys underneath the KV Prefix (defaults to 
conductor/services/*)
* These key names are assumed to be Consul service names that you want proxied
* The values for these keys need to be mount points (URL prefixes if you prefer)
* Conductor fires up background processes that watch the consul the healthy nodes
for each service
* ~~If new services are added to the KV Prefix it will notice this and send new 
traffic to those services~~ (not yet)
* If nodes become unhealthy or new nodes are added, Conductor reconfigures itself

Load Testing
============

The load testing script will fire up a group of servers, register them in consul
and then run siege against conductor.

```
boot2docker start
brew install siege
docker built -t conductor .
./loadtest.sh
```
