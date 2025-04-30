package publishkey

import (
	"context"
	"fmt"
	"net/url"

	"github.com/machinebox/graphql"
)

const graphqlOperation = `
mutation PublishAPublicKey($publicKey: String!){
  registerPublicKey(publicKey: $publicKey) {
    body
    id
  }
}
`

type Response struct {
	RegisterPublicKey struct {
		Body string `json:"body"`
		ID   string `json:"id"`
	} `json:"registerPublicKey"`
}

func (p keyPublisher) getOriginHeader() (string, error) {
	urlInfo, err := url.Parse(p.graphQLEndpoint)
	if err != nil {
		return "", err
	}

	if urlInfo.Port() == "" {
		return fmt.Sprintf("%s://%s", urlInfo.Scheme, urlInfo.Host), nil
	}

	return fmt.Sprintf("%s://%s:%s", urlInfo.Scheme, urlInfo.Host, urlInfo.Port()), nil
}

func (p keyPublisher) PublishToKeyManagementService(publicKey string) (string, error) {
	// Publish and get an ID.
	request := graphql.NewRequest(graphqlOperation)
	request.Var("publicKey", publicKey)

	origin, err := p.getOriginHeader()
	if err != nil {
		return "", err
	}

	request.Header.Add("Origin", origin)

	response, err := run[Response](graphql.NewClient(p.graphQLEndpoint), request)
	if err != nil {
		return "", err
	}

	return response.RegisterPublicKey.ID, nil
}

func run[T any](client *graphql.Client, request *graphql.Request) (*T, error) {
	response := new(T)

	err := client.Run(context.Background(), request, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
