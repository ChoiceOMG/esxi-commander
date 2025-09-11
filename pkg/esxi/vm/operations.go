package vm

import (
	"context"
	"fmt"
	
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/r11/esxi-commander/pkg/esxi/client"
)

type Operations struct {
	client *client.ESXiClient
}

type CreateOptions struct {
	Name      string
	Template  string
	CPU       int
	Memory    int // MB
	Disk      int // GB
	Guestinfo map[string]string
}

func NewOperations(c *client.ESXiClient) *Operations {
	return &Operations{client: c}
}

func (o *Operations) CreateFromTemplate(ctx context.Context, opts *CreateOptions) (*object.VirtualMachine, error) {
	template, err := o.client.FindVM(ctx, opts.Template)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	pool, err := o.client.DefaultResourcePool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource pool: %w", err)
	}

	ds, err := o.client.DefaultDatastore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get datastore: %w", err)
	}

	poolRef := pool.Reference()
	dsRef := ds.Reference()
	cloneSpec := types.VirtualMachineCloneSpec{
		Location: types.VirtualMachineRelocateSpec{
			Pool:      &poolRef,
			Datastore: &dsRef,
		},
		PowerOn:  false,
		Template: false,
	}

	if len(opts.Guestinfo) > 0 {
		var extraConfig []types.BaseOptionValue
		for key, value := range opts.Guestinfo {
			extraConfig = append(extraConfig, &types.OptionValue{
				Key:   key,
				Value: value,
			})
		}

		cloneSpec.Config = &types.VirtualMachineConfigSpec{
			NumCPUs:     int32(opts.CPU),
			MemoryMB:    int64(opts.Memory),
			ExtraConfig: extraConfig,
		}
	}

	folder, err := o.client.DefaultFolder(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	task, err := template.Clone(ctx, folder, opts.Name, cloneSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to start clone: %w", err)
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	return object.NewVirtualMachine(o.client.Client(), info.Result.(types.ManagedObjectReference)), nil
}

func (o *Operations) CloneVM(ctx context.Context, sourceName, destName string, guestinfo map[string]string) (*object.VirtualMachine, error) {
	source, err := o.client.FindVM(ctx, sourceName)
	if err != nil {
		return nil, fmt.Errorf("source VM not found: %w", err)
	}

	state, err := source.PowerState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get power state: %w", err)
	}

	if state == types.VirtualMachinePowerStatePoweredOn {
		return nil, fmt.Errorf("source VM must be powered off for cold clone")
	}

	pool, err := o.client.DefaultResourcePool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource pool: %w", err)
	}

	ds, err := o.client.DefaultDatastore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get datastore: %w", err)
	}

	poolRef := pool.Reference()
	dsRef := ds.Reference()
	cloneSpec := types.VirtualMachineCloneSpec{
		Location: types.VirtualMachineRelocateSpec{
			Pool:      &poolRef,
			Datastore: &dsRef,
		},
		PowerOn:  false,
		Template: false,
	}

	if len(guestinfo) > 0 {
		var extraConfig []types.BaseOptionValue
		for key, value := range guestinfo {
			extraConfig = append(extraConfig, &types.OptionValue{
				Key:   key,
				Value: value,
			})
		}
		cloneSpec.Config = &types.VirtualMachineConfigSpec{
			ExtraConfig: extraConfig,
		}
	}

	folder, err := o.client.DefaultFolder(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	task, err := source.Clone(ctx, folder, destName, cloneSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to start clone: %w", err)
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	return object.NewVirtualMachine(o.client.Client(), info.Result.(types.ManagedObjectReference)), nil
}

func (o *Operations) PowerOn(ctx context.Context, vm *object.VirtualMachine) error {
	task, err := vm.PowerOn(ctx)
	if err != nil {
		return fmt.Errorf("failed to start power on: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return fmt.Errorf("power on failed: %w", err)
	}

	return nil
}

func (o *Operations) Delete(ctx context.Context, name string) error {
	vm, err := o.client.FindVM(ctx, name)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	state, err := vm.PowerState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get power state: %w", err)
	}

	if state == types.VirtualMachinePowerStatePoweredOn {
		task, _ := vm.PowerOff(ctx)
		task.WaitForResult(ctx, nil)
	}

	task, err := vm.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to start destroy: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return fmt.Errorf("destroy failed: %w", err)
	}

	return nil
}

func (o *Operations) GetVMInfo(ctx context.Context, name string) (*client.VM, error) {
	vm, err := o.client.FindVM(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("VM not found: %w", err)
	}

	var mvm mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{"summary"}, &mvm)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	return &client.VM{
		Name:   mvm.Summary.Config.Name,
		UUID:   mvm.Summary.Config.Uuid,
		Status: string(mvm.Summary.Runtime.PowerState),
		IP:     mvm.Summary.Guest.IpAddress,
		CPU:    int(mvm.Summary.Config.NumCpu),
		Memory: int(mvm.Summary.Config.MemorySizeMB / 1024),
	}, nil
}
