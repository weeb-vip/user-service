package keypair

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

const KeySize = 2048

func GenerateKeyPair() (*key, error) { // nolint
	// There's no need to handle the error since rsa.GenerateKey with the given parameter can never produce error.
	keyPair, _ := rsa.GenerateKey(rand.Reader, KeySize)

	privateKey, err := getEncodedPrivateKey(keyPair)
	if err != nil {
		return nil, err
	}

	publicKey, err := getEncodedPublicKey(keyPair)
	if err != nil {
		return nil, err
	}

	return &key{PublicKey: publicKey, PrivateKey: privateKey}, nil
}

func getEncodedPrivateKey(keyPair *rsa.PrivateKey) (string, error) {
	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(keyPair)
	if err != nil {
		return "", err
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: pkcs8Key})), nil
}

func getEncodedPublicKey(keyPair *rsa.PrivateKey) (string, error) {
	publicKey := keyPair.Public()

	marshalledPublicKey, err := x509.MarshalPKIXPublicKey(publicKey.(*rsa.PublicKey))
	if err != nil {
		return "", err
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: marshalledPublicKey})), nil
}
