package types

type EndpointService struct {
	ApiInterface string
	Addon        string
}

func (es *EndpointService) String() string {
	st := es.ApiInterface
	if es.Addon != "" {
		st = st + " " + es.Addon
	}
	return st
}

func (endpoint *Endpoint) IsSupportedService(apiInterface string, addon string) bool {
	for _, endpointApiInterface := range endpoint.ApiInterfaces {
		// always support the base addons
		if addon == "" && endpointApiInterface == apiInterface {
			return true
		}
		if addon != "" {
			// only chekc addons if the requested addon is not empty
			for _, endpointAddon := range endpoint.Addons {
				if addon == endpointAddon && endpointApiInterface == apiInterface {
					return true
				}
			}
		}
	}
	return false
}

// users are allowed to stake with an empty list of apiInterfaces, this code supports adding the default ones in that case
func (endpoint *Endpoint) SetDefaultApiInterfaces(requiredServices map[EndpointService]struct{}) {
	if len(endpoint.ApiInterfaces) != 0 {
		return
	}

	for endpointService := range requiredServices {
		if endpointService.Addon != "" {
			// only required ones are taken into account
			continue
		}
		endpoint.ApiInterfaces = append(endpoint.ApiInterfaces, endpointService.ApiInterface)
	}
}

// users are allowed to send apiInterfaces inside the addons list, this code supports that
func (endpoint *Endpoint) SetApiInterfacesFromAddons(allowedServices map[EndpointService]struct{}) {
	newAddons := []string{}
	existingApiInterfaces := map[string]struct{}{}
	for _, apiInterface := range endpoint.ApiInterfaces {
		existingApiInterfaces[apiInterface] = struct{}{}
	}
	for _, addon := range endpoint.Addons {
		if _, ok := existingApiInterfaces[addon]; ok {
			continue
		}
		service := EndpointService{
			ApiInterface: addon, // we check if the addon is actually an ApiInterface
			Addon:        "",    // intentionally don't set the addon here to check if it's an api interface
		}
		if _, ok := allowedServices[service]; ok {
			// means this is an apiInterface and not an addon
			endpoint.ApiInterfaces = append(endpoint.ApiInterfaces, addon)
		} else {
			newAddons = append(newAddons, addon)
		}
	}
	endpoint.Addons = newAddons
}

func (endpoint *Endpoint) GetSupportedServices() (services []EndpointService) {
	for _, apiInterface := range endpoint.ApiInterfaces {
		// always support the base addons
		service := EndpointService{
			ApiInterface: apiInterface,
			Addon:        "",
		}
		services = append(services, service)

		for _, addon := range endpoint.Addons {
			if addon == apiInterface {
				// will be used to remove an apiInterface from the base supported
				continue
			}
			service := EndpointService{
				ApiInterface: apiInterface,
				Addon:        addon,
			}
			services = append(services, service)
		}
	}
	return services
}

func (stakeEntry *StakeEntry) GetEndpointsSupportingService(apiInterface string, addon string) (endpoints []*Endpoint) {
	for idx, endpoint := range stakeEntry.Endpoints {
		if endpoint.IsSupportedService(apiInterface, addon) {
			endpoints = append(endpoints, &stakeEntry.Endpoints[idx])
		}
	}
	return endpoints
}

func (stakeEntry *StakeEntry) GetSupportedServices() (services []EndpointService) {
	existing := map[EndpointService]struct{}{}
	for _, endpoint := range stakeEntry.Endpoints {
		for _, apiInterface := range endpoint.ApiInterfaces {
			// always support the base addons
			service := EndpointService{
				ApiInterface: apiInterface,
				Addon:        "",
			}
			if _, ok := existing[service]; !ok {
				services = append(services, service)
			}
			existing[service] = struct{}{}

			for _, addon := range endpoint.Addons {
				if addon == apiInterface {
					// will be used to remove an apiInterface from the base supported
					continue
				}
				service := EndpointService{
					ApiInterface: apiInterface,
					Addon:        addon,
				}
				if _, ok := existing[service]; !ok {
					services = append(services, service)
				}
				existing[service] = struct{}{}
			}
		}
	}
	return services
}
