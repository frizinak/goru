// +build prod

package main

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"time"

	"github.com/coreos/go-systemd/activation"
	"github.com/frizinak/gotls/tls"
)

const Prod = true

//go:embed private_account_key
var dataAccountKey []byte

//go:embed private_domain_key
var dataDomainKey []byte

func accountKey() *rsa.PrivateKey {
	c, err := x509.ParsePKCS1PrivateKey(dataAccountKey)
	if err != nil {
		panic(err)
	}
	return c
}

func domainKey() *ecdsa.PrivateKey {
	c, err := x509.ParseECPrivateKey(dataDomainKey)
	if err != nil {
		panic(err)
	}
	return c
}

func run(s *tls.Server, addr string) error {
	listeners := []interface{}{HTTPAddr, HTTPSAddr}
	_listeners, _ := activation.Listeners()
	for i := range _listeners {
		listeners[i] = _listeners[i]
	}
	return s.StartCertified(
		listeners[1],
		listeners[0],
		ACMEDir,
		Domains,
		Contact,
		time.Hour*24*30,
		accountKey(),
		domainKey(),
		CacheDir,
	)
}
