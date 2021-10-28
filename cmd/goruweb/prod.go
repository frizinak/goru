//+build prod

package main

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"time"

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
	return s.StartCertified(
		HTTPSAddr,
		HTTPAddr,
		ACMEDir,
		Domains,
		Contact,
		time.Hour*24*30,
		accountKey(),
		domainKey(),
		CacheDir,
	)
}
