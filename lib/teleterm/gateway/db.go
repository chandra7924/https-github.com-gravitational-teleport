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

package gateway

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/gravitational/trace"

	"github.com/gravitational/teleport/api/utils/keys"
	"github.com/gravitational/teleport/lib/srv/alpnproxy"
)

func (g *Gateway) makeLocalProxyForDB(listener net.Listener) error {
	tlsCert, err := keys.LoadX509KeyPair(g.cfg.CertPath, g.cfg.KeyPath)
	if err != nil {
		return trace.Wrap(err)
	}

	if err := checkCertSubject(tlsCert, g.RouteToDatabase()); err != nil {
		return trace.Wrap(err,
			"database certificate check failed, try restarting the database connection")
	}

	localProxyConfig := alpnproxy.LocalProxyConfig{
		InsecureSkipVerify:      g.cfg.Insecure,
		RemoteProxyAddr:         g.cfg.WebProxyAddr,
		Listener:                listener,
		ParentContext:           g.closeContext,
		Certs:                   []tls.Certificate{tlsCert},
		Clock:                   g.cfg.Clock,
		ALPNConnUpgradeRequired: g.cfg.TLSRoutingConnUpgradeRequired,
	}

	if g.cfg.OnExpiredCert != nil {
		localProxyConfig.Middleware = &dbMiddleware{
			log:     g.cfg.Log,
			dbRoute: g.cfg.RouteToDatabase(),
			onExpiredCert: func(ctx context.Context) error {
				err := g.cfg.OnExpiredCert(ctx, g)
				return trace.Wrap(err)
			},
		}
	}

	g.localProxy, err = alpnproxy.NewLocalProxy(localProxyConfig,
		alpnproxy.WithDatabaseProtocol(g.cfg.Protocol),
		alpnproxy.WithClusterCAsIfConnUpgrade(g.closeContext, g.cfg.RootClusterCACertPoolFunc),
	)
	return trace.Wrap(err)
}
