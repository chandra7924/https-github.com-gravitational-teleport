/*
Copyright 2023 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugins

import (
	"context"
	"sync"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"

	"github.com/gravitational/teleport"
	"github.com/gravitational/teleport/api/types"
	apievents "github.com/gravitational/teleport/api/types/events"
	commonTeleport "github.com/gravitational/teleport/integrations/access/common/teleport"
	"github.com/gravitational/teleport/integrations/lib"
	"github.com/gravitational/teleport/integrations/lib/watcherjob"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/services"
)

// Config provides configuration for the plugins server.
type Config struct {
	// Emitter is events emitter, used to submit discrete events
	Emitter apievents.Emitter
	// AccessPoint is a plugins access point
	AccessPoint auth.PluginsAccessPoint
	// Log is the logger.
	Log logrus.FieldLogger
	// APIClient is the the teleport client
	APIClient commonTeleport.Client

	// ResourceMatchers are the labels to match against plugin resources
	ResourceMatchers []services.ResourceMatcher

	*lib.Process
}

func (c *Config) CheckAndSetDefaults() error {
	if c.APIClient == nil {
		return trace.BadParameter("plugin service config missing Teleport API client")
	}
	if len(c.ResourceMatchers) == 0 {
		return trace.BadParameter("plugin service config missing missing matchers for plugins")
	}
	return nil
}

// Server is a plugins server, used to discover cloud resources for
// inclusion in Teleport
type Server struct {
	*Config

	ctx context.Context
	// cancelfn is used with ctx when stopping the plugins server
	cancelfn context.CancelFunc
	// accessRequestWatcher is an access request watcher.
	accessRequestWatcher *services.AccessRequestWatcher
	// pluginWatcher watches plugin resources.
	pluginWatcher *services.PluginWatcher

	// monitoredPlugins contains all plugins
	monitoredPlugins monitoredPlugins
	// reconcileCh triggers reconciliation of plugins.
	reconcileCh chan struct{}
	// mu protects access to  plugins.
	mu sync.RWMutex

	// runningAccessPlugins contains currently registered plugins
	// maps plugin type to RunningAccessPlugins
	runningPlugins map[string]RunningAccessPlugin
}

// New initializes a plugins Server
func New(ctx context.Context, cfg *Config) (*Server, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	localCtx, cancelfn := context.WithCancel(ctx)
	s := &Server{
		Config:           cfg,
		ctx:              localCtx,
		cancelfn:         cancelfn,
		reconcileCh:      make(chan struct{}),
		monitoredPlugins: monitoredPlugins{},
	}
	return s, nil
}

// Start starts the plugins service.
func (s *Server) Start(ctx context.Context) error {
	watcher, err := s.startResourceWatcher(ctx)
	if err != nil {
		return trace.Wrap(err)
	}
	s.pluginWatcher = watcher
	if err := s.startReconciler(ctx); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

// Stop stops the plugins service.
func (s *Server) Stop() {
	s.cancelfn()
}

// Wait will block while the server is running.
func (s *Server) Wait() error {
	<-s.ctx.Done()
	if err := s.ctx.Err(); err != nil && err != context.Canceled {
		return trace.Wrap(err)
	}
	return nil
}

func (s *Server) initTeleportAccessRequestWatcher(ctx context.Context) (err error) {
	watcherJob := watcherjob.NewJob(
		s.APIClient,
		watcherjob.Config{
			Watch: types.Watch{Kinds: []types.WatchKind{{Kind: types.KindAccessRequest}}},
		},
		s.onWatcherEvent,
	)
	s.SpawnCriticalJob(watcherJob)

	if _, err := watcherJob.WaitReady(ctx); err != nil {
		return trace.Wrap(err)
	}

	<-watcherJob.Done()

	return trace.Wrap(watcherJob.Err())
}

func (s *Server) onWatcherEvent(ctx context.Context, event types.Event) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	errs := []error{}
	for _, plugin := range s.runningPlugins {
		errs = append(errs, plugin.onWatcherEvent(ctx, event))
	}
	return trace.NewAggregate(errs...)
}

// startReconciler starts reconciler that registers/unregisters
// plugins according to the up-to-date list of plugin resources.
func (s *Server) startReconciler(ctx context.Context) error {
	reconciler, err := services.NewReconciler(services.ReconcilerConfig{
		Matcher:             s.matcher,
		GetCurrentResources: s.getResources,
		GetNewResources:     s.monitoredPlugins.get,
		OnCreate:            s.onCreate,
		OnUpdate:            s.onUpdate,
		OnDelete:            s.onDelete,
		Log:                 s.Log,
	})
	if err != nil {
		return trace.Wrap(err)
	}
	go func() {
		for {
			select {
			case <-s.reconcileCh:
				if err := reconciler.Reconcile(ctx); err != nil {
					s.Log.WithError(err).Error("Failed to reconcile.")
				}
			case <-ctx.Done():
				s.Log.Debug("Reconciler done.")
				return
			}
		}
	}()
	return nil
}

// startResourceWatcher starts watching changes to plugin resources and
// registers/unregisters the plugins accordingly.
func (s *Server) startResourceWatcher(ctx context.Context) (*services.PluginWatcher, error) {
	if len(s.Config.ResourceMatchers) == 0 {
		s.Log.Debug("Not starting plugin resource watcher.")
		return nil, nil
	}
	s.Log.Debug("Starting plugin resource watcher.")
	watcher, err := services.NewPluginWatcher(ctx, services.PluginWatcherConfig{
		ResourceWatcherConfig: services.ResourceWatcherConfig{
			Component: teleport.ComponentPlugins,
			Log:       s.Log,
			Client:    s.APIClient,
		},
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	go func() {
		defer s.Log.Debug("Plugin resource watcher done.")
		defer watcher.Close()
		for {
			select {
			case plugins := <-watcher.PluginsC:
				s.monitoredPlugins.setResources(plugins)
				select {
				case s.reconcileCh <- struct{}{}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return watcher, nil
}

// getResources returns plugins as resources.
func (s *Server) getResources() types.ResourcesWithLabelsMap {
	return s.monitoredPlugins.get()
}

// onCreate is called by reconciler when a new plugin is created.
func (s *Server) onCreate(ctx context.Context, resource types.ResourceWithLabels) error {
	plugin, ok := resource.(types.Plugin)
	if !ok {
		return trace.BadParameter("expected types.Plugin, got %T", resource)
	}
	return s.registerPlugin(ctx, plugin)
}

// onUpdate is called by reconciler when an already plugin is updated.
func (s *Server) onUpdate(ctx context.Context, resource types.ResourceWithLabels) error {
	plugin, ok := resource.(types.Plugin)
	if !ok {
		return trace.BadParameter("expected types.Plugin, got %T", resource)
	}
	return s.updatePlugin(ctx, plugin)
}

// onDelete is called by reconciler when a plugin is deleted.
func (s *Server) onDelete(ctx context.Context, resource types.ResourceWithLabels) error {
	plugin, ok := resource.(types.Plugin)
	if !ok {
		return trace.BadParameter("expected types.Plugin, got %T", resource)
	}
	return s.unregisterPlugin(ctx, plugin)
}

// matcher is used by reconciler to check if plugin matches selectors.
func (s *Server) matcher(resource types.ResourceWithLabels) bool {
	plugin, ok := resource.(types.Plugin)
	if !ok {
		return false
	}
	// Plugin resources created via CLI, or API are
	// filtered by resource matchers.
	return services.MatchResourceLabels(s.Config.ResourceMatchers, plugin)
}

func (s *Server) registerPlugin(ctx context.Context, resource types.ResourceWithLabels) error {
	plugin, ok := resource.(types.Plugin)
	if !ok {
		return trace.BadParameter("expected types.Plugin, got %T", resource)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rap, err := newRunningAccessPlugin(plugin)
	if err != nil {
		return trace.Wrap(err)
	}
	s.runningPlugins[string(plugin.GetType())] = rap
	return nil
}

func (s *Server) updatePlugin(ctx context.Context, resource types.ResourceWithLabels) error {
	if err := s.unregisterPlugin(ctx, resource); err != nil {
		return trace.Wrap(err)
	}
	if err := s.registerPlugin(ctx, resource); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

func (s *Server) unregisterPlugin(ctx context.Context, resource types.ResourceWithLabels) error {
	plugin, ok := resource.(types.Plugin)
	if !ok {
		return trace.BadParameter("expected types.Plugin, got %T", resource)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rap, ok := s.runningPlugins[string(plugin.GetType())]
	if !ok {
		return trace.BadParameter("plugin type %q is not registered", plugin.GetType())
	}
	if err := rap.Stop(ctx); err != nil {
		return trace.Wrap(err)
	}
	delete(s.runningPlugins, string(plugin.GetType()))
	return nil
}

// monitoredPlugins is a collection of plugins.
type monitoredPlugins struct {
	// resources are plugins created via CLI or API.
	resources types.Plugins
	// mu protects access to the fields.
	mu sync.RWMutex
}

func (m *monitoredPlugins) setResources(plugins types.Plugins) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = plugins
}

func (m *monitoredPlugins) get() types.ResourcesWithLabelsMap {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resources.AsResources().ToMap()
}

// RunningAccessPlugin defines the methods access plugins must implement
type RunningAccessPlugin interface {
	// Start starts an access plugin
	Start(context.Context) error
	// Stop stops an access plugin
	Stop(context.Context) error
	// OnWatcherEvent is used to pass access request events to the plugin
	onWatcherEvent(context.Context, types.Event) error
}

func newRunningAccessPlugin(plugin types.Plugin) (RunningAccessPlugin, error) {
	// Each plugin type that is needs to add a case here.
	switch plugin.GetType() {
	default:
	}
	return nil, trace.BadParameter("unknown plugin type: %q", plugin.GetType())
}
