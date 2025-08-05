package xdsclient

import (
	"google.golang.org/grpc/internal/xds/bootstrap"

	genericxdsclient "google.golang.org/grpc/xds/internal/clients/xdsclient"

	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource/version"
	xdsrc "google.org/grpc/xds/internal/xdsclient/xdsresource"
)

// supportedResourceTypes returns a map of all supported xDS resource types
// for the client. The key is the resource's TypeURL.
//
// This function is now updated to return the generic `xdsrc.Type` struct,
// directly populating it with the appropriate decoders. This removes the old
// wrapper-based `xdsclient.ResourceType` definitions.
func supportedResourceTypes(config *bootstrap.Config, gServerCfgMap map[genericxdsclient.ServerConfig]*bootstrap.ServerConfig) map[string]xdsrc.Type {
	return map[string]xdsrc.Type{
		version.V3ListenerURL: xdsrc.Type{ // Explicitly qualify with xdsrc.Type
			TypeURL:                    version.V3ListenerURL,
			TypeName:                   xdsrc.ListenerResourceTypeName,
			AllResourcesRequiredInSotW: true,
			Decoder:                    xdsrc.NewGenericListenerResourceTypeDecoder(config),
		},
		version.V3RouteConfigURL: xdsrc.Type{ // Explicitly qualify
			TypeURL:                    version.V3RouteConfigURL,
			TypeName:                   xdsrc.RouteConfigTypeName,
			AllResourcesRequiredInSotW: false,
			Decoder:                    xdsrc.NewGenericRouteConfigResourceTypeDecoder(),
		},
		version.V3ClusterURL: xdsrc.Type{ // Explicitly qualify
			TypeURL:                    version.V3ClusterURL,
			TypeName:                   xdsrc.ClusterResourceTypeName,
			AllResourcesRequiredInSotW: true,
			Decoder:                    xdsrc.NewGenericClusterResourceTypeDecoder(config, gServerCfgMap),
		},
		version.V3EndpointsURL: xdsrc.Type{ // Explicitly qualify
			TypeURL:                    version.V3EndpointsURL,
			TypeName:                   xdsrc.EndpointsResourceTypeName,
			AllResourcesRequiredInSotW: false,
			Decoder:                    xdsrc.NewGenericEndpointsResourceTypeDecoder(),
		},
	}
}
