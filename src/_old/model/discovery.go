package model

import (
	"github.com/futurehomeno/fimpgo/discovery"
)

// GetDiscoveryResource contains the discovery response
func GetDiscoveryResource() discovery.Resource {

	return discovery.Resource{
		ResourceName:           "easee",
		ResourceType:           discovery.ResourceTypeAd,
		ResourceFullName:       "Easee",
		Description:            "EV chargers from Easee",
		Author:                 "skardal@hey.com",
		IsInstanceConfigurable: false,
		InstanceId:             "1",
		Version:                "1",
		AdapterInfo: discovery.AdapterInfo{
			Technology:            "easee",
			FwVersion:             "all",
			NetworkManagementType: "inclusion_exclusion",
		},
	}

}
