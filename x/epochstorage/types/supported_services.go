package types

import "fmt"

type EndpointService struct {
	ApiInterface string
	Addon        string
	Extension    string
}

func (es *EndpointService) String() string {
	st := es.ApiInterface
	if es.Addon != "" {
		st = st + " " + es.Addon
	}
	return st
}

func (endpoint *Endpoint) isSupportedExtension(extension string) bool {
	if extension == "" {
		return true
	}
	if len(endpoint.Extensions) > 0 {
		for _, endpointExtension := range endpoint.Extensions {
			if endpointExtension == extension {
				return true
			}
		}
	}
	return false
}

func (endpoint *Endpoint) IsSupportedService(apiInterface string, addon string, extension string) bool {
	for _, endpointApiInterface := range endpoint.ApiInterfaces {
		// always support the base addons
		if addon == "" && endpointApiInterface == apiInterface {
			if endpoint.isSupportedExtension(extension) {
				return true
			}
		} else if addon != "" {
			// only check addons if the requested addon is not empty
			for _, endpointAddon := range endpoint.Addons {
				if endpointAddon == addon && endpointApiInterface == apiInterface {
					if endpoint.isSupportedExtension(extension) {
						return true
					}
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
func (endpoint *Endpoint) SetServicesFromAddons(allowedServices map[EndpointService]struct{}, addons map[string]struct{}, extensions map[string]struct{}) error {
	newAddons := []string{}
	newExtensions := []string{}
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
			Extension:    "",
		}
		if _, ok := allowedServices[service]; ok {
			// means this is an apiInterface and not an addon
			endpoint.ApiInterfaces = append(endpoint.ApiInterfaces, addon)
		} else {
			// it's not an apiInterface
			if _, ok := addons[addon]; ok {
				newAddons = append(newAddons, addon)
			} else if _, ok := extensions[addon]; ok {
				newExtensions = append(newExtensions, addon)
			} else {
				// not an addon, not an extension, not an apiInterface
				return fmt.Errorf("unknown service %s not found in spec services %+v", addon, allowedServices)
			}
		}
	}
	endpoint.Addons = newAddons
	endpoint.Extensions = append(endpoint.Extensions, newExtensions...)
	return nil
}

func (endpoint *Endpoint) GetSupportedServices() (services []EndpointService) {
	seen := map[EndpointService]struct{}{}
	for _, apiInterface := range endpoint.ApiInterfaces {
		// always support the base addons
		addons := append([]string{""}, endpoint.Addons...)
		// always support no extensions
		extensions := append([]string{""}, endpoint.Extensions...)
		for _, addon := range addons {
			if addon == apiInterface {
				// will be used to remove an apiInterface from the base supported
				continue
			}
			for _, extension := range extensions {
				service := EndpointService{
					ApiInterface: apiInterface,
					Addon:        addon,
					Extension:    extension,
				}
				if _, ok := seen[service]; !ok {
					services = append(services, service)
					seen[service] = struct{}{}
				}
			}
		}
	}
	return services
}

// can be used for both addon and a single extension
func (stakeEntry *StakeEntry) GetEndpointsSupportingService(apiInterface string, addon string, extension string) (endpoints []*Endpoint) {
	for idx, endpoint := range stakeEntry.Endpoints {
		if endpoint.IsSupportedService(apiInterface, addon, extension) {
			endpoints = append(endpoints, &stakeEntry.Endpoints[idx])
		}
	}
	return endpoints
}

func (stakeEntry *StakeEntry) GetSupportedServices() (services []EndpointService) {
	existing := map[EndpointService]struct{}{}
	for _, endpoint := range stakeEntry.Endpoints {
		endpointServices := endpoint.GetSupportedServices()
		for _, service := range endpointServices {
			if _, ok := existing[service]; !ok {
				existing[service] = struct{}{}
				services = append(services, service)
			}
		}
	}
	return services
}
