// SPDX-License-Identifier: LGPL-3.0-or-later

package vsphere

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"

	"hypersdk/config"
	"hypersdk/logger"
)

type VSphereClient struct {
	client *govmomi.Client
	finder *find.Finder
	config *config.Config
	logger logger.Logger
}

func NewVSphereClient(ctx context.Context, cfg *config.Config, log logger.Logger) (*VSphereClient, error) {
	// Parse vCenter URL
	u, err := soap.ParseURL(cfg.VCenterURL)
	if err != nil {
		return nil, fmt.Errorf("parse vCenter URL: %w", err)
	}

	// Set credentials
	u.User = url.UserPassword(cfg.Username, cfg.Password)

	// Create client with timeout
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	// Setup TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
	}

	// Create SOAP client
	soapClient := soap.NewClient(u, cfg.Insecure)
	soapClient.DefaultTransport().TLSClientConfig = tlsConfig

	// Create vSphere client
	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, fmt.Errorf("create vim25 client: %w", err)
	}

	// Create govmomi client
	client := &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}

	// Login
	if err := client.Login(ctx, u.User); err != nil {
		return nil, fmt.Errorf("login to vCenter: %w", err)
	}

	// Create finder
	finder := find.NewFinder(client.Client, true)

	// Find default datacenter
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		log.Warn("no default datacenter found, using first available", "error", err)
		// Try to find any datacenter
		dcs, dcErr := finder.DatacenterList(ctx, "*")
		if dcErr != nil || len(dcs) == 0 {
			return nil, fmt.Errorf("find datacenter: %w", err)
		}
		dc = dcs[0]
	}

	finder.SetDatacenter(dc)

	log.Info("connected to vSphere",
		"url", cfg.VCenterURL,
		"datacenter", dc.Name())

	return &VSphereClient{
		client: client,
		finder: finder,
		config: cfg,
		logger: log,
	}, nil
}

func (c *VSphereClient) Close() error {
	if c.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return c.client.Logout(ctx)
	}
	return nil
}
