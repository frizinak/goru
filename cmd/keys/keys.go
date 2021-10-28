package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	generate(os.Args[1], os.Args[2])
	// generate("rsa", "cmd/goruweb/private_account_key")
	// generate("ecdsa", "cmd/goruweb/private_domain_key")
}

func generate(which, path string) {
	if _, err := os.Stat(path); err == nil {
		panic("key " + path + " exists")
	}
	var err error
	switch which {
	case "rsa":
		_, err = generateRSAKey(
			path,
			4096,
		)
	case "ecdsa":
		_, err = generateECDSAKey(
			path,
			elliptic.P256(),
		)
	default:
		log.Fatal("no such algo")
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Println("done")
}

func generateRSAKey(filepath string, bits int) (*rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}

	marsh := x509.MarshalPKCS1PrivateKey(priv)
	err = ioutil.WriteFile(filepath, marsh, 0600)

	return priv, err
}

func generateECDSAKey(filepath string, curve elliptic.Curve) (*ecdsa.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	marsh, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(filepath, marsh, 0600)

	return priv, err
}
