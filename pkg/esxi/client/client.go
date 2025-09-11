package client

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type ESXiClient struct {
	client     *govmomi.Client
	datacenter *object.Datacenter
	finder     *find.Finder
	config     *Config
}

type Config struct {
	Host     string
	User     string
	Password string
	Insecure bool
	Timeout  time.Duration
}

type VM struct {
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Status string `json:"status"`
	IP     string `json:"ip,omitempty"`
	CPU    int    `json:"cpu"`
	Memory int    `json:"memory"` // GB
}

func NewClient(cfg *Config) (*ESXiClient, error) {
	ctx := context.Background()

	u, err := soap.ParseURL(fmt.Sprintf("https://%s/sdk", cfg.Host))
	if err != nil {
		return nil, fmt.Errorf("invalid host URL: %w", err)
	}

	u.User = url.UserPassword(cfg.User, cfg.Password)

	soapClient := soap.NewClient(u, cfg.Insecure)

	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create vim25 client: %w", err)
	}

	c := &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}

	if err := c.Login(ctx, u.User); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	f := find.NewFinder(c.Client, true)

	dc, err := f.DefaultDatacenter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get datacenter: %w", err)
	}

	f.SetDatacenter(dc)

	return &ESXiClient{
		client:     c,
		datacenter: dc,
		finder:     f,
		config:     cfg,
	}, nil
}

func (c *ESXiClient) Client() *vim25.Client {
	return c.client.Client
}

func (c *ESXiClient) Finder() *find.Finder {
	return c.finder
}

func (c *ESXiClient) FindVM(ctx context.Context, name string) (*object.VirtualMachine, error) {
	vm, err := c.finder.VirtualMachine(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' not found: %w", name, err)
	}
	return vm, nil
}

func (c *ESXiClient) DefaultResourcePool(ctx context.Context) (*object.ResourcePool, error) {
	pool, err := c.finder.DefaultResourcePool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default resource pool: %w", err)
	}
	return pool, nil
}

func (c *ESXiClient) DefaultDatastore(ctx context.Context) (*object.Datastore, error) {
	ds, err := c.finder.DefaultDatastore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default datastore: %w", err)
	}
	return ds, nil
}

func (c *ESXiClient) DefaultFolder(ctx context.Context) (*object.Folder, error) {
	folder, err := c.finder.DefaultFolder(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default folder: %w", err)
	}
	return folder, nil
}

func (c *ESXiClient) DefaultHost(ctx context.Context) (*object.HostSystem, error) {
	hosts, err := c.finder.HostSystemList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts found")
	}
	return hosts[0], nil
}

func (c *ESXiClient) ListVMs(ctx context.Context) ([]*VM, error) {
	vms, err := c.finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	var result []*VM
	for _, vm := range vms {
		var mvm mo.VirtualMachine
		err := vm.Properties(ctx, vm.Reference(), []string{"summary"}, &mvm)
		if err != nil {
			continue
		}

		result = append(result, &VM{
			Name:   mvm.Summary.Config.Name,
			UUID:   mvm.Summary.Config.Uuid,
			Status: string(mvm.Summary.Runtime.PowerState),
			IP:     mvm.Summary.Guest.IpAddress,
			CPU:    int(mvm.Summary.Config.NumCpu),
			Memory: int(mvm.Summary.Config.MemorySizeMB / 1024),
		})
	}

	return result, nil
}

// GetHostSystem returns the host system (alias for DefaultHost)
func (c *ESXiClient) GetHostSystem(ctx context.Context) (*object.HostSystem, error) {
	return c.DefaultHost(ctx)
}

// RetrieveOne retrieves properties for a single managed object
func (c *ESXiClient) RetrieveOne(ctx context.Context, ref types.ManagedObjectReference, ps []string, dst interface{}) error {
	pc := c.client.Client.ServiceContent.PropertyCollector
	return mo.RetrieveProperties(ctx, c.client.Client, pc, ref, dst)
}

func (c *ESXiClient) Close() error {
	if c.client != nil {
		return c.client.Logout(context.Background())
	}
	return nil
}
