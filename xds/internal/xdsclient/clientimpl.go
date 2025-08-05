package xdsclient

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	estats "google.golang.org/grpc/experimental/stats"
	"google.golang.org/grpc/internal/grpclog"
	"google.golang.org/grpc/internal/xds/bootstrap"

	// This is where ResourceUpdate is now defined.
	genericxdsclient "google.golang.org/grpc/xds/internal/clients/xdsclient"

	"google.golang.org/grpc/xds/internal/clients/lrsclient"
	"google.golang.org/grpc/xds/internal/xdsclient/transport"
	"google.golang.org/grpc/xds/internal/xdsclient/xdsresource"
	// Import the version package
)

// Constants, vars, and metricsReporter remain the same.
const (
	// defaultWatchExpiryTimeout is the timeout for xDS watches. If a watch
	// does not receive an update from the management server within this
	// duration, it will be considered expired and an error will be reported
	// to the watcher.
	defaultWatchExpiryTimeout = 15 * time.Minute
)

var (
	logger = grpclog.Component("xds")
	// For testing.
	metricsReporter = func(_ string) estats.MetricsReporter { return nil }
)

// clientImpl is the concrete implementation of the XDSClient interface.
type clientImpl struct {
	// embed for default methods (e.g. for testing default xdsclient without affecting grpc.ClientConn.xdsClient)
	// No, we are removing the embedding for the new generic client.

	// xdsClient that contains the real implementation of all xDS methods.
	// We are removing this field. The xdsclient.Client is now the lower-level interface.

	// We are not maintaining a pointer to the xDSClient here anymore.
	// clientImpl itself implements the XDSClient interface.

	// This is the new internal generic xDS client.
	// This field will represent the lower-level transport and state management.
	// internalClient genericxdsclient.Client // Moved to transportManager for direct interaction

	// refCount is an atomic counter that records the number of active
	// clientConn's that are using this xdsClient.
	refCount int32

	// Configuration
	config *bootstrap.Config // The bootstrap configuration

	// transportManager handles the actual xDS stream management and proto decoding.
	transportManager transport.Manager

	// Watcher management
	mu sync.Mutex
	// watchers is a map from resource type to a map of resource name to watcherInfo.
	// This stores the currently active watchers for each resource type.
	watchers      map[xdsresource.Type]map[string]*watcherInfo
	lastResources map[xdsresource.Type]map[string]interface{}

	// supportedResources stores the map of supported resource types,
	// populated from resource_types.go. This is used to retrieve the full
	// xdsresource.Type struct when a watch is initiated.
	supportedResources map[string]xdsresource.Type
}

// watcherInfo holds the channel and cancellation function for a specific watch.
// The channel now uses the aliased generic ResourceUpdate type.
type watcherInfo struct {
	updateCh chan genericxdsclient.ResourceUpdate
	cancel   context.CancelFunc
}

// newClientImpl creates a new xDS client implementation.
// It initializes the internal components like transport, backoff, and watcher maps.
func newClientImpl(config *bootstrap.Config) *clientImpl {
	c := &clientImpl{
		config:           config,
		transportManager: transport.NewManager(config), // Initialize transport manager
		watchers:         make(map[xdsresource.Type]map[string]*watcherInfo),
		lastResources:    make(map[xdsresource.Type]map[string]interface{}),
	}

	c.supportedResources = supportedResourceTypes(config, nil) // gServerCfgMap can be nil for now or passed from outer context
	return c
}

// Close decrements the refCount and closes the client if refCount becomes zero.
func (c *clientImpl) Close() {
	if atomic.AddInt32(&c.refCount, -1) == 0 {
		c.transportManager.Close()
	}
}

// BootstrapConfig returns the bootstrap configuration used by this client.
func (c *clientImpl) BootstrapConfig() *bootstrap.Config {
	return c.config
}

// ReportLoad is used to report load to the xDS management server.
// This method forwards the call to the transport manager.
func (c *clientImpl) ReportLoad(scfg *bootstrap.ServerConfig) (*lrsclient.LoadStore, func(context.Context)) {
	return c.transportManager.ReportLoad(scfg)
}

// The watcher methods implement the XDSClient interface directly.
func (c *clientImpl) WatchListener(resourceName string, watcher ListenerWatcher) (cancel func()) {
	// Retrieve the full xdsresource.Type for Listener
	rType, ok := c.supportedResources[xdsresource.V3ListenerURL]
	if !ok {
		err := fmt.Errorf("unknown resource type URL: %s", xdsresource.V3ListenerURL)
		watcher.OnError(resourceName, err)
		return func() {}
	}
	return c.watch(rType, resourceName, func(resource interface{}, err error) {
		if err != nil {
			watcher.OnError(resourceName, err)
			return
		}
		listener, ok := resource.(*xdsresource.Listener)
		if !ok {
			c.logger.Errorf("Internal error: received non-Listener resource for Listener watcher: %T", resource)
			watcher.OnError(resourceName, fmt.Errorf("internal error: unexpected resource type received"))
			return
		}
		watcher.OnUpdate(resourceName, listener)
	})
}

func (c *clientImpl) WatchRouteConfig(resourceName string, watcher RouteConfigWatcher) (cancel func()) {
	// Retrieve the full xdsresource.Type for RouteConfig
	rType, ok := c.supportedResources[xdsresource.V3RouteConfigURL]
	if !ok {
		err := fmt.Errorf("unknown resource type URL: %s", xdsresource.V3RouteConfigURL)
		watcher.OnError(resourceName, err)
		return func() {}
	}
	return c.watch(rType, resourceName, func(resource interface{}, err error) {
		if err != nil {
			watcher.OnError(resourceName, err)
			return
		}
		routeConfig, ok := resource.(*xdsresource.RouteConfig)
		if !ok {
			c.logger.Errorf("Internal error: received non-RouteConfig resource for RouteConfig watcher: %T", resource)
			watcher.OnError(resourceName, fmt.Errorf("internal error: unexpected resource type received"))
			return
		}
		watcher.OnUpdate(resourceName, routeConfig)
	})
}

func (c *clientImpl) WatchCluster(resourceName string, watcher ClusterWatcher) (cancel func()) {
	// Retrieve the full xdsresource.Type for Cluster
	rType, ok := c.supportedResources[xdsresource.V3ClusterURL]
	if !ok {
		err := fmt.Errorf("unknown resource type URL: %s", xdsresource.V3ClusterURL)
		watcher.OnError(resourceName, err)
		return func() {}
	}
	return c.watch(rType, resourceName, func(resource interface{}, err error) {
		if err != nil {
			watcher.OnError(resourceName, err)
			return
		}
		cluster, ok := resource.(*xdsresource.Cluster)
		if !ok {
			c.logger.Errorf("Internal error: received non-Cluster resource for Cluster watcher: %T", resource)
			watcher.OnError(resourceName, fmt.Errorf("internal error: unexpected resource type received"))
			return
		}
		watcher.OnUpdate(resourceName, cluster)
	})
}

func (c *clientImpl) WatchEndpoints(resourceName string, watcher EndpointsWatcher) (cancel func()) {
	// Retrieve the full xdsresource.Type for Endpoints
	rType, ok := c.supportedResources[xdsresource.V3EndpointsURL]
	if !ok {
		err := fmt.Errorf("unknown resource type URL: %s", xdsresource.V3EndpointsURL)
		watcher.OnError(resourceName, err)
		return func() {}
	}
	return c.watch(rType, resourceName, func(resource interface{}, err error) {
		if err != nil {
			watcher.OnError(resourceName, err)
			return
		}
		endpoints, ok := resource.(*xdsresource.Endpoints)
		if !ok {
			c.logger.Errorf("Internal error: received non-Endpoints resource for Endpoints watcher: %T", resource)
			watcher.OnError(resourceName, fmt.Errorf("internal error: unexpected resource type received"))
			return
		}
		watcher.OnUpdate(resourceName, endpoints)
	})
}

// watch is a generic helper function to manage a watch for any resource type.
func (c *clientImpl) watch(rType xdsresource.Type, resourceName string, updateHandler func(resource interface{}, err error)) (cancel func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.watchers[rType]; !ok {
		c.watchers[rType] = make(map[string]*watcherInfo)
	}

	if info, ok := c.watchers[rType][resourceName]; ok {
		c.logger.Debugf("Watch already exists for resource %q of type %q, returning existing cancel func", resourceName, rType.TypeName)
		return func() {}
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	updateCh := make(chan genericxdsclient.ResourceUpdate, 1)
	info := &watcherInfo{updateCh: updateCh, cancel: cancelFunc}

	c.watchers[rType][resourceName] = info
	// The transportManager.AddWatch now receives the full xdsresource.Type.
	c.transportManager.AddWatch(rType, resourceName)

	if lastUpdates, ok := c.lastResources[rType]; ok {
		if resource, ok := lastUpdates[resourceName]; ok {
			updateHandler(resource, nil)
		}
	}

	go func() {
		defer func() {
			c.mu.Lock()
			delete(c.watchers[rType], resourceName)
			if len(c.watchers[rType]) == 0 {
				delete(c.watchers, rType)
				c.transportManager.RemoveWatchType(rType)
			}
			c.mu.Unlock()
			c.logger.Debugf("Watcher for %s/%s cancelled.", rType.TypeName, resourceName)
		}()

		timer := time.NewTimer(defaultWatchExpiryTimeout)
		for {
			select {
			case update := <-updateCh:
				timer.Reset(defaultWatchExpiryTimeout)
				if update.Err != nil {

					updateHandler(nil, update.Err)
				} else {

					if res, ok := update.Resources[resourceName]; ok {
						updateHandler(res, nil)
					}
				}
			case <-ctx.Done():
				return
			case <-timer.C:
				c.logger.Warningf("Watch for resource %q of type %q expired, no update received for %v", resourceName, rType.TypeName, defaultWatchExpiryTimeout)
				updateHandler(nil, fmt.Errorf("xDS watch expired for %s/%s", rType.TypeName, resourceName))
				timer.Reset(defaultWatchExpiryTimeout)
			}
		}
	}()

	return func() {
		cancelFunc()
	}
}

// handleResourceUpdate is the callback from the transport layer.
func (c *clientImpl) handleResourceUpdate(update genericxdsclient.ResourceUpdate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if update.Err == nil {
		if _, ok := c.lastResources[update.Type]; !ok {
			c.lastResources[update.Type] = make(map[string]interface{})
		}
		for name, resource := range update.Resources {
			c.lastResources[update.Type][name] = resource
		}
	}

	watchMap, ok := c.watchers[update.Type]
	if !ok {
		c.logger.Warningf("Received update for un-watched resource type %q", update.Type.TypeName)
		return
	}

	// Dispatch updates to all watchers for this resource type.
	for name, resource := range update.Resources {
		if wi, found := watchMap[name]; found {
			// Send the resource update to the watcher's channel.
			wi.updateCh <- genericxdsclient.ResourceUpdate{
				Type: update.Type,
				Resources: map[string]interface{}{
					name: resource,
				},
				Err: nil,
			}
		}
	}

	// handling the errors
	// The generic client might send an update with a nil Resources map and an error.
	if update.Err != nil {
		for _, wi := range watchMap {
			wi.updateCh <- genericxdsclient.ResourceUpdate{
				Type:      update.Type,
				Resources: nil,
				Err:       update.Err,
			}
		}
	}
}

// prefixLogger is a helper to create a consistent logger for the xDS client.
func prefixLogger(prefix string) *grpclog.PrefixLogger {
	return grpclog.NewPrefixLogger(logger, fmt.Sprintf("[%s] ", prefix))
}
