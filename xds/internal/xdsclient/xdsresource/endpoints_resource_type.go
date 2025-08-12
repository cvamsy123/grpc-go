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

	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"google.golang.org/grpc/internal/xds/bootstrap"
	xdsclient "google.golang.org/grpc/xds/internal/clients/xdsclient"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// EndpointsResourceTypeName represents the transport agnostic name for the
	// endpoints resource.
	EndpointsResourceTypeName = "EndpointsResource"
)

// Endpoints represents the internal data structure for an Endpoints resource.
type Endpoints struct {
	*v3endpointpb.ClusterLoadAssignment
}

func (e *Endpoints) Bytes() []byte {
	b, _ := proto.Marshal(e.ClusterLoadAssignment)
	return b
}
func (e *Endpoints) Raw() *anypb.Any {
	any, _ := anypb.New(e.ClusterLoadAssignment)
	return any
}

// Equal returns true if other is equal to r.
func (e *Endpoints) Equal(other xdsclient.ResourceData) bool {
	o, ok := other.(*Endpoints)
	if !ok {
		return false
	}
	return proto.Equal(e.ClusterLoadAssignment, o.ClusterLoadAssignment)
}
func (e *Endpoints) ToJSON() string         { return e.ClusterLoadAssignment.String() }
func (e *Endpoints) ResourceName() string   { return e.GetClusterName() }
func (e *Endpoints) GetClusterName() string { return e.ClusterLoadAssignment.ClusterName }

// EndpointsTypeImpl provides the resource-type specific functionality for a
// ClusterLoadAssignment resource.
//
// Implements the Type interface and xdsclient.Decoder.
type EndpointsTypeImpl struct {
	resourceTypeState
	BootstrapConfig *bootstrap.Config
	ServerConfigMap map[xdsclient.ServerConfig]*bootstrap.ServerConfig
}

var endpointsType = EndpointsTypeImpl{
	resourceTypeState: resourceTypeState{
		typeURL:                    version.V3EndpointsURL,
		typeName:                   "Endpoints",
		allResourcesRequiredInSotW: false,
	},
}

func (et EndpointsTypeImpl) TypeURL() string {
	return et.resourceTypeState.TypeURL()
}
func (et EndpointsTypeImpl) TypeName() string {
	return et.resourceTypeState.TypeName()
}
func (et EndpointsTypeImpl) AllResourcesRequiredInSotW() bool {
	return et.resourceTypeState.AllResourcesRequiredInSotW()
}

// Decode deserializes and validates an xDS resource serialized inside the
// provided `Any` proto, as received from the xDS management server.
func (et EndpointsTypeImpl) Decode(resource xdsclient.AnyProto, gOpts xdsclient.DecodeOptions) (*xdsclient.DecodeResult, error) {
	anyProto := &anypb.Any{TypeUrl: resource.TypeURL, Value: resource.Value}

	opts := &DecodeOptions{BootstrapConfig: et.BootstrapConfig}
	if gOpts.ServerConfig != nil {
		if bootstrapSC, ok := et.ServerConfigMap[*gOpts.ServerConfig]; ok {
			opts.ServerConfig = bootstrapSC
		} else {
			return nil, fmt.Errorf("xdsresource: server config %v not found in map", *gOpts.ServerConfig)
		}
	}

	var endpointsProto v3endpointpb.ClusterLoadAssignment
	if err := anyProto.UnmarshalTo(&endpointsProto); err != nil {
		return nil, fmt.Errorf("xdsresource: failed to unmarshal Endpoints: %v", err)
	}

	xdsClientResourceData := &Endpoints{ClusterLoadAssignment: &endpointsProto}

	return &xdsclient.DecodeResult{
		Name:     endpointsProto.GetClusterName(),
		Resource: xdsClientResourceData,
	}, nil
}

// EndpointsTypeURL returns the type URL for Endpoints resources.
func EndpointsTypeURL() string {
	return version.V3EndpointsURL
}
