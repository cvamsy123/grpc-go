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

	"google.golang.org/grpc/internal/pretty"
	"google.golang.org/grpc/internal/xds/bootstrap"
	xdsclient "google.golang.org/grpc/xds/internal/clients/xdsclient"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// EndpointsResourceTypeName represents the transport agnostic name for the
	// endpoint resource.
	EndpointsResourceTypeName = "EndpointsResource"
)

// endpointsResourceType provides the resource-type specific functionality for a
// ClusterLoadAssignment (or Endpoints) resource.
//
// Implements the Type interface.
type endpointsResourceType struct {
	resourceTypeState
}

// Decode deserializes and validates an xDS resource serialized inside the
// provided `Any` proto, as received from the xDS management server.
func (endpointsResourceType) Decode(_ *DecodeOptions, resource *anypb.Any) (*DecodeResult, error) {
	name, rc, err := unmarshalEndpointsResource(resource)
	switch {
	case name == "":
		// Name is unset only when protobuf deserialization fails.
		return nil, err
	case err != nil:
		// Protobuf deserialization succeeded, but resource validation failed.
		return &DecodeResult{Name: name, Resource: &EndpointsResourceData{Resource: EndpointsUpdate{}}}, err
	}

	return &DecodeResult{Name: name, Resource: &EndpointsResourceData{Resource: rc}}, nil

}

// EndpointsResourceData wraps the configuration of an Endpoints resource as
// received from the management server.
//
// Implements the ResourceData interface.
type EndpointsResourceData struct {
	ResourceData

	// TODO: We have always stored update structs by value. See if this can be
	// switched to a pointer?
	Resource EndpointsUpdate
}

// RawEqual returns true if other is equal to r.
func (e *EndpointsResourceData) RawEqual(other ResourceData) bool {
	if e == nil && other == nil {
		return true
	}
	if (e == nil) != (other == nil) {
		return false
	}
	return proto.Equal(e.Resource.Raw, other.Raw())

}

// ToJSON returns a JSON string representation of the resource data.
func (e *EndpointsResourceData) ToJSON() string {
	return pretty.ToJSON(e.Resource)
}

// Raw returns the underlying raw protobuf form of the listener resource.
func (e *EndpointsResourceData) Raw() *anypb.Any {
	return e.Resource.Raw
}

// EndpointsWatcher wraps the callbacks to be invoked for different
// events corresponding to the endpoints resource being watched. gRFC A88
// contains an exhaustive list of what method is invoked under what conditions.
type EndpointsWatcher interface {
	ResourceChanged(resources map[string]*Endpoints)
	ResourceError(err error)
	AmbientError(err error)
}

type delegatingEndpointsWatcher struct {
	watcher EndpointsWatcher
}

func (d *delegatingEndpointsWatcher) ResourceChanged(resources map[string]ResourceData) {
	if d.watcher == nil {
		return
	}
	ret := make(map[string]*Endpoints, len(resources))
	for name, res := range resources {
		ret[name] = res.(*Endpoints)
	}
	d.watcher.ResourceChanged(ret)
}

func (d *delegatingEndpointsWatcher) ResourceError(err error) {
	if d.watcher == nil {
		return
	}
	d.watcher.ResourceError(err)
}

func (d *delegatingEndpointsWatcher) AmbientError(err error) {
	if d.watcher == nil {
		return
	}
	d.watcher.AmbientError(err)
}

// WatchEndpoints uses xDS to discover the configuration associated with the
// provided endpoints resource name.
func WatchEndpoints(p Producer, name string, w EndpointsWatcher) (cancel func()) {
	delegator := &delegatingEndpointsWatcher{watcher: w}
	return p.WatchResource(endpointsType, name, delegator)
}

// NewGenericEndpointsResourceTypeDecoder returns a xdsclient.Decoder that
// wraps the xdsresource.endpointsType.
func NewGenericEndpointsResourceTypeDecoder(bootstrapConfig *bootstrap.Config, m map[xdsclient.ServerConfig]*bootstrap.ServerConfig) xdsclient.Decoder {
	et := endpointsTypeImpl{
		resourceTypeState: resourceTypeState{
			typeURL:                    version.V3EndpointsURL,
			typeName:                   "Endpoints",
			allResourcesRequiredInSotW: false,
		},
		bootstrapConfig: bootstrapConfig,
		serverConfigMap: m,
	}

	return &GenericResourceTypeDecoder{
		decoder: func(resource xdsclient.AnyProto, gOpts xdsclient.DecodeOptions) (*xdsclient.DecodeResult, error) {
			anyProto := &anypb.Any{TypeUrl: resource.TypeURL, Value: resource.Value}
			opts := &DecodeOptions{BootstrapConfig: et.bootstrapConfig}

			if gOpts.ServerConfig != nil {
				if bootstrapSC, ok := et.serverConfigMap[*gOpts.ServerConfig]; ok {
					opts.ServerConfig = bootstrapSC
				} else {
					return nil, fmt.Errorf("xdsresource: server config %v not found in map", *gOpts.ServerConfig)
				}
			}

			internalResult, err := et.Decode(opts, anyProto)
			if err != nil {
				if internalResult != nil {
					return &xdsclient.DecodeResult{Name: internalResult.Name}, err
				}
				return nil, err
			}
			if internalResult == nil {
				return nil, fmt.Errorf("xdsresource: internal decode returned nil result but no error")
			}
			xdsClientResourceData, ok := internalResult.Resource.(xdsclient.ResourceData)
			if !ok {
				return nil, fmt.Errorf("xdsresource: internal resource of type %T does not implement xdsclient.ResourceData", internalResult.Resource)
			}
			return &xdsclient.DecodeResult{Name: internalResult.Name, Resource: xdsClientResourceData}, nil
		},
	}
}
