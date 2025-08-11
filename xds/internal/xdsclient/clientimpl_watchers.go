/*
 *
 * Copyright 2020 gRPC authors.
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

package xdsclient

import (
	"google.golang.org/grpc/xds/internal/clients/xdsclient"
	xdsresource "google.golang.org/grpc/xds/internal/xdsclient/xdsresource"
	"google.golang.org/protobuf/types/known/anypb"
)

// WatchResource uses xDS to discover the resource associated with the provided
// resource name. The resource type implementation determines how xDS responses
// are are deserialized and validated, as received from the xDS management
// server. Upon receipt of a response from the management server, an
// appropriate callback on the watcher is invoked.
type ClientImpl struct {
	xdsclient.XDSClient
}

type ResourceData interface {
	ResourceName() string
	Equal(other ResourceData) bool
	ToJSON() string
	Raw() *anypb.Any
	Bytes() []byte
}

func (c *ClientImpl) WatchResource(rType xdsresource.Type, resourceName string, watcher xdsresource.ResourceWatcher) (cancel func()) {
	// The resource watcher here implements xdsclient.ResourceWatcher.
	// It converts the generic xdsclient.ResourceData to the specific
	// xdsresource.ResourceData.
	return c.XDSClient.WatchResource(rType.TypeURL(), resourceName, &genericResourceWatcher{
		watcher: watcher,
		name:    resourceName,
	})
}

// genericResourceWatcher is a wrapper around a xdsresource.ResourceWatcher that
// implements the xdsclient.ResourceWatcher interface, performing the necessary
// type conversions.
type genericResourceWatcher struct {
	watcher xdsresource.ResourceWatcher
	name    string
}

// ResourceChanged implements xdsclient.ResourceWatcher.
func (g *genericResourceWatcher) ResourceChanged(resource xdsclient.ResourceData, done func()) {
	defer done()
	if g.watcher == nil {
		return
	}

	ret := make(map[string]xdsresource.ResourceData, 1)
	ret[g.name] = resource.(xdsresource.ResourceData)

	g.watcher.ResourceChanged(ret)
}

// ResourceError implements xdsclient.ResourceWatcher.
func (g *genericResourceWatcher) ResourceError(err error, done func()) {
	defer done()
	if g.watcher == nil {
		return
	}
	g.watcher.ResourceError(err)
}

// AmbientError implements xdsclient.ResourceWatcher.
func (g *genericResourceWatcher) AmbientError(err error, done func()) {
	defer done()
	if g.watcher == nil {
		return
	}
	g.watcher.AmbientError(err)
}
