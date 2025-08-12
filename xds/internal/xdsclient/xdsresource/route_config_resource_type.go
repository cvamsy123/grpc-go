/*
 *
 * Copyright 2022 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xdsresource

import (
	"fmt"

	v3routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/grpc/internal/xds/bootstrap"
	xdsclient "google.golang.org/grpc/xds/internal/clients/xdsclient"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// RouteConfigTypeName represents the transport agnostic name for the
	// route config resource.
	RouteConfigTypeName = "RouteConfigResource"
)

// RouteConfig represents the internal data structure for a RouteConfig resource.
type RouteConfig struct {
	*v3routepb.RouteConfiguration
}

func (rc *RouteConfig) Bytes() []byte {
	b, _ := proto.Marshal(rc.RouteConfiguration)
	return b
}
func (rc *RouteConfig) Raw() *anypb.Any {
	any, _ := anypb.New(rc.RouteConfiguration)
	return any
}
func (rc *RouteConfig) Equal(other xdsclient.ResourceData) bool {
	o, ok := other.(*RouteConfig)
	if !ok {
		return false
	}
	return proto.Equal(rc.RouteConfiguration, o.RouteConfiguration)
}

func (rc *RouteConfig) ToJSON() string       { return rc.RouteConfiguration.String() }
func (rc *RouteConfig) ResourceName() string { return rc.GetName() }
func (rc *RouteConfig) GetName() string      { return rc.RouteConfiguration.Name }

// RouteConfigTypeImpl provides the resource-type specific functionality for a
// RouteConfiguration resource.
//
// Implements the Type interface and xdsclient.Decoder.
type RouteConfigTypeImpl struct {
	resourceTypeState
	BootstrapConfig *bootstrap.Config
	ServerConfigMap map[xdsclient.ServerConfig]*bootstrap.ServerConfig
}

var routeConfigType = RouteConfigTypeImpl{
	resourceTypeState: resourceTypeState{
		typeURL:                    version.V3RouteConfigURL,
		typeName:                   "RouteConfig",
		allResourcesRequiredInSotW: false,
	},
}

func (rct RouteConfigTypeImpl) TypeURL() string  { return rct.resourceTypeState.TypeURL() }
func (rct RouteConfigTypeImpl) TypeName() string { return rct.resourceTypeState.TypeName() }
func (rct RouteConfigTypeImpl) AllResourcesRequiredInSotW() bool {
	return rct.resourceTypeState.AllResourcesRequiredInSotW()
}

// Decode deserializes and validates an xDS resource serialized inside the
// provided `Any` proto, as received from the xDS management server.
func (rct RouteConfigTypeImpl) Decode(resource xdsclient.AnyProto, gOpts xdsclient.DecodeOptions) (*xdsclient.DecodeResult, error) {
	anyProto := &anypb.Any{TypeUrl: resource.TypeURL, Value: resource.Value}

	opts := &DecodeOptions{BootstrapConfig: rct.BootstrapConfig}
	if gOpts.ServerConfig != nil {
		if bootstrapSC, ok := rct.ServerConfigMap[*gOpts.ServerConfig]; ok {
			opts.ServerConfig = bootstrapSC
		} else {
			return nil, fmt.Errorf("xdsresource: server config %v not found in map", *gOpts.ServerConfig)
		}
	}

	var routeConfigProto v3routepb.RouteConfiguration
	if err := anyProto.UnmarshalTo(&routeConfigProto); err != nil {
		return nil, fmt.Errorf("xdsresource: failed to unmarshal RouteConfig: %v", err)
	}

	xdsClientResourceData := &RouteConfig{RouteConfiguration: &routeConfigProto}

	return &xdsclient.DecodeResult{
		Name:     routeConfigProto.GetName(),
		Resource: xdsClientResourceData,
	}, nil
}
