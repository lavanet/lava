package epoch

// EndpointService describes a single service interface offered on an endpoint.
type EndpointService struct {
	ApiInterface string `json:"api_interface"`
	Extension    string `json:"extension"`
}

func (e *EndpointService) GetApiInterface() string {
	if e == nil {
		return ""
	}
	return e.ApiInterface
}

func (e *EndpointService) GetExtension() string {
	if e == nil {
		return ""
	}
	return e.Extension
}
