/*
 *
 * Copyright 2019 gRPC authors.
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
 *
 */

// Package xdsclient implements a full fledged gRPC client for the xDS API used
// by the xds resolver and balancer implementations.
package xdsclient

import (
	"context"

	v3statuspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	"google.golang.org/grpc/internal/xds/bootstrap"
	"google.golang.org/grpc/xds/internal/clients/lrsclient"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource"
)

type ListenerWatcher interface {
	OnUpdate(resourceName string, listener *xdsresource.Listener)
	OnError(resourceName string, err error)
}

type RouteConfigWatcher interface {
	OnUpdate(resourceName string, routeConfig *xdsresource.RouteConfig)
	OnError(resourceName string, err error)
}

type ClusterWatcher interface {
	OnUpdate(resourceName string, cluster *xdsresource.Cluster)
	OnError(resourceName string, err error)
}

type EndpointsWatcher interface {
	OnUpdate(resourceName string, endpoints *xdsresource.Endpoints)
	OnError(resourceName string, err error)
}

// XDSClient is a full fledged gRPC client which queries a set of discovery APIs
// (collectively termed as xDS) on a remote management server, to discover
// various dynamic resources.
type XDSClient interface {
	// WatchResource uses xDS to discover the resource associated with the
	// provided resource name. The resource type implementation determines how
	// xDS responses are are deserialized and validated, as received from the
	// xDS management server. Upon receipt of a response from the management
	// server, an appropriate callback on the watcher is invoked.
	//
	// Most callers will not have a need to use this API directly. They will
	// instead use a resource-type-specific wrapper API provided by the relevant
	// resource type implementation.
	//
	//
	// During a race (e.g. an xDS response is received while the user is calling
	// cancel()), there's a small window where the callback can be called after
	// the watcher is canceled. Callers need to handle this case.
	WatchListener(resourceName string, watcher ListenerWatcher) (cancel func())

	// WatchRouteConfig uses xDS to discover the RouteConfiguration resource.
	WatchRouteConfig(resourceName string, watcher RouteConfigWatcher) (cancel func())

	// WatchCluster uses xDS to discover the Cluster resource.
	WatchCluster(resourceName string, watcher ClusterWatcher) (cancel func())

	// WatchEndpoints uses xDS to discover the ClusterLoadAssignment (Endpoints) resource.
	WatchEndpoints(resourceName string, watcher EndpointsWatcher) (cancel func())

	// ReportLoad is used to report load to the xDS management server.
	ReportLoad(*bootstrap.ServerConfig) (*lrsclient.LoadStore, func(context.Context))

	// BootstrapConfig returns the configuration read from the bootstrap file.
	BootstrapConfig() *bootstrap.Config

	// Close closes the xDS client, releasing all resources.
	Close()
}

// DumpResources returns the status and contents of all xDS resources. It uses
// xDS clients from the default pool.
func DumpResources() *v3statuspb.ClientStatusResponse {
	return DefaultPool.DumpResources()
}
