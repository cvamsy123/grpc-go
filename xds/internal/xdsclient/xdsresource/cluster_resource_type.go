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

	v3clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"google.golang.org/grpc/internal/xds/bootstrap"
	xdsclient "google.golang.org/grpc/xds/internal/clients/xdsclient"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// ClusterResourceTypeName represents the transport agnostic name for the
	// cluster resource.
	ClusterResourceTypeName = "ClusterResource"
)

type Cluster struct {
	*v3clusterpb.Cluster
}

func (c *Cluster) Bytes() []byte {
	b, _ := proto.Marshal(c.Cluster)
	return b
}

func (c *Cluster) Raw() *anypb.Any {
	any, _ := anypb.New(c.Cluster)
	return any
}

// Equal returns true if other is equal to r.
func (c *Cluster) Equal(other xdsclient.ResourceData) bool {
	o, ok := other.(*Cluster)
	if !ok {
		return false
	}
	return proto.Equal(c.Cluster, o.Cluster)
}

func (c *Cluster) ToJSON() string       { return c.Cluster.String() }
func (c *Cluster) ResourceName() string { return c.GetName() }
func (c *Cluster) GetName() string      { return c.Cluster.Name }

// ClusterTypeImpl provides the resource-type specific functionality for a
// Cluster resource.
//
// Implements the Type interface and xdsclient.Decoder.
type ClusterTypeImpl struct {
	resourceTypeState
	BootstrapConfig *bootstrap.Config
	ServerConfigMap map[xdsclient.ServerConfig]*bootstrap.ServerConfig
}

var clusterType = ClusterTypeImpl{
	resourceTypeState: resourceTypeState{
		typeURL:                    version.V3ClusterURL,
		typeName:                   "Cluster",
		allResourcesRequiredInSotW: false,
	},
}

func (ct ClusterTypeImpl) TypeURL() string  { return ct.resourceTypeState.TypeURL() }
func (ct ClusterTypeImpl) TypeName() string { return ct.resourceTypeState.TypeName() }
func (ct ClusterTypeImpl) AllResourcesRequiredInSotW() bool {
	return ct.resourceTypeState.AllResourcesRequiredInSotW()
}

// Decode deserializes and validates an xDS resource serialized inside the
// provided `Any` proto, as received from the xDS management server.
func (ct ClusterTypeImpl) Decode(resource xdsclient.AnyProto, gOpts xdsclient.DecodeOptions) (*xdsclient.DecodeResult, error) {
	anyProto := &anypb.Any{TypeUrl: resource.TypeURL, Value: resource.Value}

	opts := &DecodeOptions{BootstrapConfig: ct.BootstrapConfig}
	if gOpts.ServerConfig != nil {
		if bootstrapSC, ok := ct.ServerConfigMap[*gOpts.ServerConfig]; ok {
			opts.ServerConfig = bootstrapSC
		} else {
			return nil, fmt.Errorf("xdsresource: server config %v not found in map", *gOpts.ServerConfig)
		}
	}

	var clusterProto v3clusterpb.Cluster
	if err := anyProto.UnmarshalTo(&clusterProto); err != nil {
		return nil, fmt.Errorf("xdsresource: failed to unmarshal Cluster: %v", err)
	}

	xdsClientResourceData := &Cluster{Cluster: &clusterProto}

	return &xdsclient.DecodeResult{Name: clusterProto.GetName(), Resource: xdsClientResourceData}, nil
}
