package routing

import (
	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/fimpgo/fimptype"
)

// GetDiscoveryResource returns a service discovery configuration.
func GetDiscoveryResource() *discovery.Resource {
	return &discovery.Resource{
		ResourceName:           fimptype.EaseeService.Str(),
		ResourceType:           discovery.ResourceTypeAd,
		ResourceFullName:       "Easee",
		Description:            "EV chargers from Easee",
		Author:                 "support@futurehome.no",
		IsInstanceConfigurable: false,
		Version:                "1",
		InstanceID:             "1",
		AdapterInfo: discovery.AdapterInfo{
			Technology:            "easee",
			FwVersion:             "all",
			NetworkManagementType: "inclusion_exclusion",
		},
	}
}
