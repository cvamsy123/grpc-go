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

	v3listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"google.golang.org/grpc/internal/xds/bootstrap"
	xdsclient "google.golang.org/grpc/xds/internal/clients/xdsclient"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// ListenerResourceTypeName represents the transport agnostic name for the
	// listener resource.
	ListenerResourceTypeName = "ListenerResource"
)

type Listener struct {
	*v3listenerpb.Listener
}

func (l *Listener) Bytes() []byte {
	b, _ := proto.Marshal(l.Listener)
	return b
}
func (l *Listener) Raw() *anypb.Any {
	any, _ := anypb.New(l.Listener)
	return any
}

// Equal returns true if other is equal to l.
func (l *Listener) Equal(other xdsclient.ResourceData) bool {
	o, ok := other.(*Listener)
	if !ok {
		return false
	}
	return proto.Equal(l.Listener, o.Listener)
}
func (l *Listener) ToJSON() string       { return l.Listener.String() }
func (l *Listener) ResourceName() string { return l.GetName() }
func (l *Listener) GetName() string      { return l.Listener.Name }

// ListenerTypeImpl provides the resource-type specific functionality for a
// Listener resource.
//
// Implements the Type interface and xdsclient.Decoder.
type ListenerTypeImpl struct {
	resourceTypeState
	BootstrapConfig *bootstrap.Config
	ServerConfigMap map[xdsclient.ServerConfig]*bootstrap.ServerConfig
}

var listenerType = ListenerTypeImpl{
	resourceTypeState: resourceTypeState{
		typeURL:                    ListenerTypeURL(),
		typeName:                   "Listener",
		allResourcesRequiredInSotW: false,
	},
}

func (lt ListenerTypeImpl) TypeURL() string  { return lt.resourceTypeState.TypeURL() }
func (lt ListenerTypeImpl) TypeName() string { return lt.resourceTypeState.TypeName() }
func (lt ListenerTypeImpl) AllResourcesRequiredInSotW() bool {
	return lt.resourceTypeState.AllResourcesRequiredInSotW()
}

// Decode deserializes and validates an xDS resource serialized inside the
// provided `Any` proto, as received from the xDS management server.
func (lt ListenerTypeImpl) Decode(resource xdsclient.AnyProto, gOpts xdsclient.DecodeOptions) (*xdsclient.DecodeResult, error) {
	anyProto := &anypb.Any{TypeUrl: resource.TypeURL, Value: resource.Value}

	opts := &DecodeOptions{BootstrapConfig: lt.BootstrapConfig}
	if gOpts.ServerConfig != nil {
		if bootstrapSC, ok := lt.ServerConfigMap[*gOpts.ServerConfig]; ok {
			opts.ServerConfig = bootstrapSC
		} else {
			return nil, fmt.Errorf("xdsresource: server config %v not found in map", *gOpts.ServerConfig)
		}
	}

	var listenerProto v3listenerpb.Listener
	if err := anyProto.UnmarshalTo(&listenerProto); err != nil {
		return nil, fmt.Errorf("xdsresource: failed to unmarshal Listener: %v", err)
	}

	xdsClientResourceData := &Listener{Listener: &listenerProto}

	return &xdsclient.DecodeResult{
		Name:     listenerProto.GetName(),
		Resource: xdsClientResourceData,
	}, nil
}

// ListenerTypeURL returns the type URL for Listener resources.
func ListenerTypeURL() string {
	return version.V3ListenerURL
}
