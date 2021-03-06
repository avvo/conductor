package main

import (
	"fmt"
	"net/url"
)

func NewNiaveRoundRobin(s Service) func() url.URL {
	i := -1
	if len(s.Nodes) == 0 {
		return func() url.URL { return url.URL{} }
	} else {
		return func() url.URL {
			i = (i + 1) % len(s.Nodes)
			node := s.Nodes[i]
			url := url.URL{
				Host:   fmt.Sprintf("%s:%d", node.Address, node.Port),
				Scheme: "http",
			}
			return url
		}
	}
}
