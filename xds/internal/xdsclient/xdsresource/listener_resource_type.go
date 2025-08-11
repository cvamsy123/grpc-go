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

// listenerResourceType provides the resource-type specific functionality for a
// Listener resource.
//
// Implements the Type interface.

func (l *Listener) Bytes() []byte {
	b, _ := proto.Marshal(l.Listener)
	return b
}
func (l *Listener) RawEqual(other ResourceData) bool {
	o, ok := other.(*Listener)
	if !ok {
		return false
	}
	return proto.Equal(l.Listener, o.Listener)
}
func (l *Listener) Raw() *anypb.Any {
	any, _ := anypb.New(l.Listener)
	return any
}
func (l *Listener) Equal(other xdsclient.ResourceData) bool {
	o, ok := other.(*Listener)
	if !ok {
		return false
	}
	return l.RawEqual(o)
}

func (l *Listener) ToJSON() string       { return l.Listener.String() }
func (l *Listener) ResourceName() string { return l.GetName() }
func (l *Listener) GetName() string      { return l.Listener.Name }

// ListenerWatcher wraps a ResourceWatcher and provides a type-safe API.
type ListenerWatcher interface {
	ResourceChanged(resources map[string]*Listener)
	ResourceError(err error)
	AmbientError(err error)
}

// delegatingListenerWatcher is a wrapper around a ListenerWatcher that implements
// the xdsresource.ResourceWatcher interface, performing the necessary type conversions.
type delegatingListenerWatcher struct {
	watcher ListenerWatcher
}

func (d *delegatingListenerWatcher) ResourceChanged(resources map[string]ResourceData) {
	if d.watcher == nil {
		return
	}
	ret := make(map[string]*Listener, len(resources))
	for name, res := range resources {
		ret[name] = res.(*Listener)
	}
	d.watcher.ResourceChanged(ret)
}

func (d *delegatingListenerWatcher) ResourceError(err error) {
	if d.watcher == nil {
		return
	}
	d.watcher.ResourceError(err)
}

func (d *delegatingListenerWatcher) AmbientError(err error) {
	if d.watcher == nil {
		return
	}
	d.watcher.AmbientError(err)
}

func WatchListener(p Producer, name string, w ListenerWatcher) (cancel func()) {
	delegator := &delegatingListenerWatcher{watcher: w}
	return p.WatchResource(listenerType, name, delegator)
}

var listenerType = listenerTypeImpl{
	resourceTypeState: resourceTypeState{
		typeURL:                    ListenerTypeURL(),
		typeName:                   "Listener",
		allResourcesRequiredInSotW: false,
	},
}

func (lt listenerTypeImpl) TypeURL() string  { return lt.resourceTypeState.TypeURL() }
func (lt listenerTypeImpl) TypeName() string { return lt.resourceTypeState.TypeName() }
func (lt listenerTypeImpl) AllResourcesRequiredInSotW() bool {
	return lt.resourceTypeState.AllResourcesRequiredInSotW()
}

// ListenerTypeURL returns the type URL for Listener resources.
func ListenerTypeURL() string {
	return version.V3ListenerURL
}
