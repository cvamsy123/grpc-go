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
	v3routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource/version"
)

const (
	// RouteConfigTypeName represents the transport agnostic name for the
	// route config resource.
	RouteConfigTypeName = "RouteConfigResource"
)

// routeConfigResourceType provides the resource-type specific functionality for
// a RouteConfiguration resource.
//
// Implements the Type interface.
type RouteConfigResource struct {
	*v3routepb.RouteConfiguration
}

func (rc *RouteConfig) ResourceName() string { return rc.GetName() }
func (rc *RouteConfig) GetName() string      { return rc.RouteConfiguration.Name }

// RouteConfigWatcher wraps a ResourceWatcher and provides a type-safe API.
type RouteConfigWatcher interface {
	ResourceChanged(resources map[string]*RouteConfig)
	ResourceError(err error)
	AmbientError(err error)
}

// delegatingRouteConfigWatcher is a wrapper around a RouteConfigWatcher that implements
// the xdsresource.ResourceWatcher interface, performing the necessary type conversions.
type delegatingRouteConfigWatcher struct {
	watcher RouteConfigWatcher
}

func (d *delegatingRouteConfigWatcher) ResourceChanged(resources map[string]ResourceData) {
	if d.watcher == nil {
		return
	}
	ret := make(map[string]*RouteConfig, len(resources))
	for name, res := range resources {
		ret[name] = res.(*RouteConfig)
	}
	d.watcher.ResourceChanged(ret)
}

func (d *delegatingRouteConfigWatcher) ResourceError(err error) {
	if d.watcher == nil {
		return
	}
	d.watcher.ResourceError(err)
}

func (d *delegatingRouteConfigWatcher) AmbientError(err error) {
	if d.watcher == nil {
		return
	}
	d.watcher.AmbientError(err)
}

func WatchRouteConfig(p Producer, name string, w RouteConfigWatcher) (cancel func()) {
	delegator := &delegatingRouteConfigWatcher{watcher: w}
	return p.WatchResource(routeConfigType, name, delegator)
}

// RouteConfigTypeURL returns the type URL for RouteConfig resources.
func RouteConfigTypeURL() string {
	return version.V3RouteConfigURL
}

// NewRouteConfigDecoder returns a new Decoder for RouteConfig resources.
