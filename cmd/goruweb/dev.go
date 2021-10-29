// +build !prod

package main

import (
	"net/http"

	_ "net/http/pprof"

	"github.com/frizinak/gotls/tls"
)

const Prod = false

func run(s *tls.Server, addr string) error {
	go http.ListenAndServe(":6060", nil)
	return s.Start(addr, false)
}
