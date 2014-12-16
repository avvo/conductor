package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func NewReverseProxyWithLoadBalancer(mountPoint string, lbFunc func() url.URL) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = lbFunc().Host
		originalRequest := req.URL.Path
		req.URL.Path = strings.TrimPrefix(req.URL.Path, mountPoint)
		log.WithFields(log.Fields{
			"original_request":  originalRequest,
			"rewritten_request": req.URL.Path,
			"mount_point":       mountPoint,
			"forward_to":        req.URL.Host,
		}).Info("Proxying request")
	}
	return &httputil.ReverseProxy{Director: director}
}
