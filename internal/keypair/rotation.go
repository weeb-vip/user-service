package keypair

import (
	"time"

	"github.com/weeb-vip/user-service/internal/container"
)

type PublicKeyIDGenerator func(publicKey string) (string, error)

type keyRotator struct {
	keyContainer   container.Container[*key]
	keyIDGenerator PublicKeyIDGenerator
}

type RotatingSigningKey interface {
	Rotate()
	RotateInBackground(every time.Duration)
	GetLatest() SigningKey
}

func (k keyRotator) RotateInBackground(every time.Duration) {
	go func() {
		for {
			time.Sleep(every)
			k.Rotate()
		}
	}()
}

func (k keyRotator) Rotate() {
	newKeyPair, err := generateNewKeyPairWithID(k.keyIDGenerator)
	if err != nil {
		// Because it's okay to not rotate key for few times.
		return
	}

	k.keyContainer.ReplaceWith(newKeyPair)
}

func (k keyRotator) GetLatest() SigningKey {
	currentKey := k.keyContainer.GetLatest()

	return SigningKey{
		Key: currentKey.PrivateKey,
		ID:  currentKey.ID,
	}
}

func NewSigningKeyRotator(idGenerator PublicKeyIDGenerator) (RotatingSigningKey, error) {
	// We start with generating a key and keeping it in container[key].
	keyPair, err := generateNewKeyPairWithID(idGenerator)
	if err != nil {
		return nil, err
	}

	keyContainer := container.New[*key](keyPair)

	return keyRotator{
		keyContainer:   keyContainer,
		keyIDGenerator: idGenerator,
	}, nil
}

func generateNewKeyPairWithID(idGenerator PublicKeyIDGenerator) (*key, error) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	keyID, err := idGenerator(keyPair.PublicKey)
	if err != nil {
		return nil, err
	}

	keyPair.ID = keyID

	return keyPair, nil
}
