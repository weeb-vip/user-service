package publishkey

type KeyPublisher interface {
	PublishToKeyManagementService(publicKey string) (string, error)
}

type keyPublisher struct {
	graphQLEndpoint string
}

func NewKeyPublisher(graphqlEndpoint string) KeyPublisher {
	return keyPublisher{graphQLEndpoint: graphqlEndpoint}
}
