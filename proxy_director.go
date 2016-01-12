package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func NewReverseProxyWithLoadBalancer(mountPoint string, requests chan *chan url.URL) *httputil.ReverseProxy {
	response := make(chan url.URL, 1)
	director := func(req *http.Request) {
		// send our channel to the worker
		requests <- &response
		// Get the server URL as the response back on the channel
		server := <-response

		req.URL.Scheme = "http"
		req.URL.Host = server.Host
		originalRequest := req.URL.Path
		req.URL.Path = strings.TrimPrefix(req.URL.Path, mountPoint)

		if server.Host == "" {
			log.WithFields(log.Fields{
				"original_request":  originalRequest,
				"rewritten_request": req.URL.Path,
				"mount_point":       mountPoint,
				"forward_to":        req.URL.Host,
			}).Warn("No host found for this endpoint")
		}

		log.WithFields(log.Fields{
			"original_request":  originalRequest,
			"rewritten_request": req.URL.Path,
			"mount_point":       mountPoint,
			"forward_to":        req.URL.Host,
		}).Info("Proxying request")
	}

	return &httputil.ReverseProxy{Director: director}
}
