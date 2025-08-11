/*
 *
 * Copyright 2022 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.


package xdsresource

// GenericResourceWatcher is a wrapper around a ResourceWatcher that implements the generic
// watcher interface required by xdsclient.
type GenericResourceWatcher struct {
	watcher ResourceWatcher
	rType   xdsresource.Type
}

// NewGenericResourceWatcher returns a new GenericResourceWatcher.
func NewGenericResourceWatcher(watcher ResourceWatcher) *GenericResourceWatcher {
	return &GenericResourceWatcher{
		watcher: watcher,
	}
}

// Update implements xdsclient.GenericResourceWatcher.
func (g *GenericResourceWatcher) Update(resources map[string]ResourceData, done func()) {
	if g.watcher == nil {
		done()
		return
	}

	ret := make(map[string]xdsresource.ResourceData, len(resources))
	for name, res := range resources {
		ret[name] = res.(xdsresource.ResourceData)
	}
	g.watcher.Update(ret)
}

// ResourceDoesNotExist implements xdsclient.ResourceWatcher.
func (g *GenericResourceWatcher) ResourceDoesNotExist(done func()) {
	defer done()
	if g.watcher == nil {
		return
	}
	g.watcher.ResourceDoesNotExist()
}

// Error implements xdsclient.ResourceWatcher.
func (g *GenericResourceWatcher) Error(err error, done func()) {
	defer done()
	if g.watcher == nil {
		return
	}
	g.watcher.Error(err)
}

// AmbientError implements xdsclient.ResourceWatcher.
func (g *GenericResourceWatcher) AmbientError(err error, done func()) {
	defer done()
	if g.watcher == nil {
		return
	}
	g.watcher.Error(err)
}
*/