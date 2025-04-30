package entities

type ServiceError struct {
	Code     string
	Message  string
	Metadata interface{}
}

func (e ServiceError) Error() string {
	return e.Message
}
