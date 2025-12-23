// Package cryptohelpers Description: This file contains functions to generate public and private key pairs
package cryptohelpers

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/nacl/box"
)

// GenerateKeys returns a freshly generated NaCl box public & private key,
func GenerateKeys() (privatePEM, publicPEM string, err error) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}

	pubB64 := base64.StdEncoding.EncodeToString(pub[:])
	privB64 := base64.StdEncoding.EncodeToString(priv[:])

	return pubB64, privB64, nil
}

// import (
//	"crypto/ed25519"
//	"crypto/rand"
//	"crypto/x509"
//	"encoding/pem"
//)
//
//// GenerateEd25519KeyPair returns a freshly generated Ed25519 private & public key,
//// both PEM-encoded.
// func GenerateEd25519KeyPair() (privatePEM, publicPEM []byte, err error) {
//	// Generate the Ed25519 keypair
//	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	// ----- Private key -> PKCS#8 -> PEM -----
//	privBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
//	if err != nil {
//		return nil, nil, err
//	}
//	privatePEM = pem.EncodeToMemory(&pem.Block{
//		Type:  "PRIVATE KEY", // PKCS#8 generic type
//		Bytes: privBytes,
//	})
//
//	// ----- Public key -> PKIX -> PEM -----
//	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
//	if err != nil {
//		return nil, nil, err
//	}
//	publicPEM = pem.EncodeToMemory(&pem.Block{
//		Type:  "PUBLIC KEY",
//		Bytes: pubBytes,
//	})
//
//	return privatePEM, publicPEM, nil
//}

// package main
