package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
)

func main() {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	b, _ := x509.MarshalECPrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: b,
	}
	pem.Encode(os.Stdout, pemBlock)
}
