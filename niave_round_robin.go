package main

import (
	"fmt"
	"net/url"
)

func NewNiaveRoundRobin(s *Service) func() url.URL {
	i := -1
	return func() url.URL {
		i = (i + 1) % len(s.Nodes)
		node := *s.Nodes[i]
		url := url.URL{
			Host:   fmt.Sprintf("%s:%d", node.Address, s.Port),
			Scheme: "http",
		}
		return url
	}
}
