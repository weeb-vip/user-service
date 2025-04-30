package keypair

type key struct {
	PrivateKey string
	PublicKey  string
	ID         string
}

type SigningKey struct {
	Key string
	ID  string
}
