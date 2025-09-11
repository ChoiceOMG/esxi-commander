package vm

import (
	"context"
	"fmt"
	"time"
	
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/r11/esxi-commander/pkg/audit"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/metrics"
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
	start := time.Now()
	
	// Start audit logging
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.create", map[string]interface{}{
		"name":     opts.Name,
		"template": opts.Template,
		"cpu":      opts.CPU,
		"memory":   opts.Memory,
		"disk":     opts.Disk,
	})
	
	template, err := o.client.FindVM(ctx, opts.Template)
	if err != nil {
		metrics.RecordVMOperation("create", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return nil, fmt.Errorf("template not found: %w", err)
	}

	pool, err := o.client.DefaultResourcePool(ctx)
	if err != nil {
		metrics.RecordVMOperation("create", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
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
	start := time.Now()
	
	source, err := o.client.FindVM(ctx, sourceName)
	if err != nil {
		metrics.RecordVMOperation("clone", "failure", time.Since(start).Seconds())
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
		metrics.RecordVMOperation("clone", "failure", time.Since(start).Seconds())
		return nil, fmt.Errorf("failed to start clone: %w", err)
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		metrics.RecordVMOperation("clone", "failure", time.Since(start).Seconds())
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	metrics.RecordVMOperation("clone", "success", time.Since(start).Seconds())
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
	start := time.Now()
	
	vm, err := o.client.FindVM(ctx, name)
	if err != nil {
		metrics.RecordVMOperation("delete", "failure", time.Since(start).Seconds())
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
		metrics.RecordVMOperation("delete", "failure", time.Since(start).Seconds())
		return fmt.Errorf("failed to start destroy: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		metrics.RecordVMOperation("delete", "failure", time.Since(start).Seconds())
		return fmt.Errorf("destroy failed: %w", err)
	}

	metrics.RecordVMOperation("delete", "success", time.Since(start).Seconds())
	return nil
}

// PowerOff powers off a VM (hard power off)
func (o *Operations) PowerOff(ctx context.Context, vm *object.VirtualMachine) error {
	start := time.Now()
	
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.poweroff", map[string]interface{}{
		"vm": vm.Name(),
	})
	
	task, err := vm.PowerOff(ctx)
	if err != nil {
		metrics.RecordVMOperation("poweroff", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("failed to start power off: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		metrics.RecordVMOperation("poweroff", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("power off failed: %w", err)
	}

	metrics.RecordVMOperation("poweroff", "success", time.Since(start).Seconds())
	auditCtx.Success()
	return nil
}

// Shutdown gracefully shuts down a VM (requires VMware Tools)
func (o *Operations) Shutdown(ctx context.Context, vm *object.VirtualMachine) error {
	start := time.Now()
	
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.shutdown", map[string]interface{}{
		"vm": vm.Name(),
	})
	
	err := vm.ShutdownGuest(ctx)
	if err != nil {
		metrics.RecordVMOperation("shutdown", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("failed to shutdown guest: %w", err)
	}

	metrics.RecordVMOperation("shutdown", "success", time.Since(start).Seconds())
	auditCtx.Success()
	return nil
}

// Restart restarts a VM
func (o *Operations) Restart(ctx context.Context, vm *object.VirtualMachine) error {
	start := time.Now()
	
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.restart", map[string]interface{}{
		"vm": vm.Name(),
	})

	// Try graceful restart first
	err := vm.RebootGuest(ctx)
	if err != nil {
		// Fall back to hard restart
		task, err := vm.Reset(ctx)
		if err != nil {
			metrics.RecordVMOperation("restart", "failure", time.Since(start).Seconds())
			auditCtx.Failure(err)
			return fmt.Errorf("failed to restart VM: %w", err)
		}
		_, err = task.WaitForResult(ctx, nil)
		if err != nil {
			metrics.RecordVMOperation("restart", "failure", time.Since(start).Seconds())
			auditCtx.Failure(err)
			return fmt.Errorf("failed to restart VM: %w", err)
		}
	}

	metrics.RecordVMOperation("restart", "success", time.Since(start).Seconds())
	auditCtx.Success()
	return nil
}

// Suspend suspends a VM
func (o *Operations) Suspend(ctx context.Context, vm *object.VirtualMachine) error {
	start := time.Now()
	
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.suspend", map[string]interface{}{
		"vm": vm.Name(),
	})
	
	task, err := vm.Suspend(ctx)
	if err != nil {
		metrics.RecordVMOperation("suspend", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("failed to start suspend: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		metrics.RecordVMOperation("suspend", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("suspend failed: %w", err)
	}

	metrics.RecordVMOperation("suspend", "success", time.Since(start).Seconds())
	auditCtx.Success()
	return nil
}

// Resume resumes a suspended VM (alias for PowerOn)
func (o *Operations) Resume(ctx context.Context, vm *object.VirtualMachine) error {
	return o.PowerOn(ctx, vm)
}

// GetPowerState returns the current power state of a VM
func (o *Operations) GetPowerState(ctx context.Context, vm *object.VirtualMachine) (types.VirtualMachinePowerState, error) {
	return vm.PowerState(ctx)
}

// CreateSnapshot creates a snapshot of a VM
func (o *Operations) CreateSnapshot(ctx context.Context, vm *object.VirtualMachine, name, description string, memory, quiesce bool) error {
	start := time.Now()
	
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.snapshot.create", map[string]interface{}{
		"vm":          vm.Name(),
		"name":        name,
		"memory":      memory,
		"quiesce":     quiesce,
	})
	
	task, err := vm.CreateSnapshot(ctx, name, description, memory, quiesce)
	if err != nil {
		metrics.RecordVMOperation("snapshot.create", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		metrics.RecordVMOperation("snapshot.create", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("snapshot creation failed: %w", err)
	}

	metrics.RecordVMOperation("snapshot.create", "success", time.Since(start).Seconds())
	auditCtx.Success()
	return nil
}

// RemoveSnapshot removes a snapshot
func (o *Operations) RemoveSnapshot(ctx context.Context, vm *object.VirtualMachine, snapshotRef types.ManagedObjectReference, removeChildren bool) error {
	start := time.Now()
	
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.snapshot.remove", map[string]interface{}{
		"vm":               vm.Name(),
		"snapshot":         snapshotRef.Value,
		"remove_children":  removeChildren,
	})
	
	task, err := vm.RemoveSnapshot(ctx, snapshotRef.Value, removeChildren, nil)
	if err != nil {
		metrics.RecordVMOperation("snapshot.remove", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("failed to remove snapshot: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		metrics.RecordVMOperation("snapshot.remove", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("snapshot removal failed: %w", err)
	}

	metrics.RecordVMOperation("snapshot.remove", "success", time.Since(start).Seconds())
	auditCtx.Success()
	return nil
}

// RevertToSnapshot reverts VM to a snapshot
func (o *Operations) RevertToSnapshot(ctx context.Context, vm *object.VirtualMachine, snapshotRef types.ManagedObjectReference) error {
	start := time.Now()
	
	auditCtx := audit.GetLogger().LogOperation(ctx, "vm.snapshot.revert", map[string]interface{}{
		"vm":       vm.Name(),
		"snapshot": snapshotRef.Value,
	})
	
	task, err := vm.RevertToSnapshot(ctx, snapshotRef.Value, true)
	if err != nil {
		metrics.RecordVMOperation("snapshot.revert", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("failed to revert to snapshot: %w", err)
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		metrics.RecordVMOperation("snapshot.revert", "failure", time.Since(start).Seconds())
		auditCtx.Failure(err)
		return fmt.Errorf("snapshot revert failed: %w", err)
	}

	metrics.RecordVMOperation("snapshot.revert", "success", time.Since(start).Seconds())
	auditCtx.Success()
	return nil
}

// ListSnapshots lists all snapshots for a VM
func (o *Operations) ListSnapshots(ctx context.Context, vm *object.VirtualMachine) ([]types.VirtualMachineSnapshotTree, error) {
	var vmObj mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{"snapshot"}, &vmObj)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	if vmObj.Snapshot == nil {
		return []types.VirtualMachineSnapshotTree{}, nil
	}

	return vmObj.Snapshot.RootSnapshotList, nil
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
