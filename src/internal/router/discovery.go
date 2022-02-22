package router

import (
	"github.com/futurehomeno/cliffhanger/discovery"

	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
)

// GetDiscoveryResource returns a service discovery configuration.
func GetDiscoveryResource() *discovery.Resource {
	return &discovery.Resource{
		ResourceName:           easee.ServiceName,
		ResourceType:           discovery.ResourceTypeAd,
		ResourceFullName:       "Easee",
		Description:            "EV chargers from Easee",
		Author:                 "support@futurehome.no",
		IsInstanceConfigurable: false,
		Version:                "1",
		AdapterInfo: discovery.AdapterInfo{
			Technology:            "easee",
			FwVersion:             "all",
			NetworkManagementType: "inclusion_exclusion",
		},
	}
}
