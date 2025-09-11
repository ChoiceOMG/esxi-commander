package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/r11/esxi-commander/pkg/config"
	"github.com/r11/esxi-commander/pkg/esxi/client"
	"github.com/r11/esxi-commander/pkg/esxi/vm"
	"github.com/spf13/cobra"
	"github.com/vmware/govmomi/vim25/types"
)

var snapshotFlags struct {
	memory      bool
	quiesce     bool
	description string
	children    bool
}

// NewSnapshotCommand creates the VM snapshot management command
func NewSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "VM snapshot management",
		Long:  `Manage VM snapshots: create, list, revert, and delete.`,
	}

	cmd.AddCommand(
		NewSnapshotCreateCommand(),
		NewSnapshotListCommand(),
		NewSnapshotRevertCommand(),
		NewSnapshotDeleteCommand(),
	)

	return cmd
}

// NewSnapshotCreateCommand creates a snapshot
func NewSnapshotCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <vm-name> <snapshot-name>",
		Short: "Create a VM snapshot",
		Long: `Create a snapshot of a virtual machine.

The snapshot captures the VM's current state including:
- Virtual disk contents
- VM configuration
- Optionally: Memory state (with --memory)
- Optionally: File system consistency (with --quiesce)`,
		Args: cobra.ExactArgs(2),
		RunE: runSnapshotCreate,
	}

	cmd.Flags().BoolVar(&snapshotFlags.memory, "memory", false, "Include memory state in snapshot")
	cmd.Flags().BoolVar(&snapshotFlags.quiesce, "quiesce", false, "Quiesce VM file system (requires VMware Tools)")
	cmd.Flags().StringVar(&snapshotFlags.description, "description", "", "Snapshot description")

	return cmd
}

// NewSnapshotListCommand lists snapshots
func NewSnapshotListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <vm-name>",
		Short: "List VM snapshots",
		Long:  `List all snapshots for a virtual machine.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSnapshotList,
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

// NewSnapshotRevertCommand reverts to a snapshot
func NewSnapshotRevertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revert <vm-name> <snapshot-name>",
		Short: "Revert VM to snapshot",
		Long: `Revert a virtual machine to a specific snapshot.

This will restore the VM to the exact state it was in when the snapshot was created,
including disk contents, configuration, and optionally memory state.`,
		Args: cobra.ExactArgs(2),
		RunE: runSnapshotRevert,
	}

	return cmd
}

// NewSnapshotDeleteCommand deletes a snapshot
func NewSnapshotDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <vm-name> <snapshot-name>",
		Short: "Delete a VM snapshot",
		Long:  `Delete a snapshot from a virtual machine.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runSnapshotDelete,
	}

	cmd.Flags().BoolVar(&snapshotFlags.children, "children", false, "Also delete child snapshots")

	return cmd
}

func runSnapshotCreate(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	snapshotName := args[1]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	clientConfig := &client.Config{
		Host:     cfg.ESXi.Host,
		User:     cfg.ESXi.User,
		Password: cfg.ESXi.Password,
		Insecure: cfg.ESXi.Insecure,
	}

	esxiClient, err := client.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	ops := vm.NewOperations(esxiClient)
	ctx := context.Background()

	vmObj, err := esxiClient.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	fmt.Printf("Creating snapshot '%s' for VM '%s'...\n", snapshotName, vmName)
	if snapshotFlags.memory {
		fmt.Println("Including memory state in snapshot")
	}
	if snapshotFlags.quiesce {
		fmt.Println("Quiescing file system (VMware Tools required)")
	}

	description := snapshotFlags.description
	if description == "" {
		description = fmt.Sprintf("Snapshot created on %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	err = ops.CreateSnapshot(ctx, vmObj, snapshotName, description, snapshotFlags.memory, snapshotFlags.quiesce)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	fmt.Printf("Snapshot '%s' created successfully\n", snapshotName)
	return nil
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	vmName := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	clientConfig := &client.Config{
		Host:     cfg.ESXi.Host,
		User:     cfg.ESXi.User,
		Password: cfg.ESXi.Password,
		Insecure: cfg.ESXi.Insecure,
	}

	esxiClient, err := client.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	ops := vm.NewOperations(esxiClient)
	ctx := context.Background()

	vmObj, err := esxiClient.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	snapshots, err := ops.ListSnapshots(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Printf("No snapshots found for VM '%s'\n", vmName)
		return nil
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		output, _ := json.MarshalIndent(snapshots, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Snapshots for VM '%s':\n", vmName)
		printSnapshotTree(snapshots, 0)
	}

	return nil
}

func printSnapshotTree(snapshots []types.VirtualMachineSnapshotTree, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, snap := range snapshots {
		created := snap.CreateTime.Format("2006-01-02 15:04:05")
		fmt.Printf("%s- %s (created: %s)\n", prefix, snap.Name, created)
		if snap.Description != "" {
			fmt.Printf("%s  Description: %s\n", prefix, snap.Description)
		}
		if len(snap.ChildSnapshotList) > 0 {
			printSnapshotTree(snap.ChildSnapshotList, indent+1)
		}
	}
}

func runSnapshotRevert(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	snapshotName := args[1]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	clientConfig := &client.Config{
		Host:     cfg.ESXi.Host,
		User:     cfg.ESXi.User,
		Password: cfg.ESXi.Password,
		Insecure: cfg.ESXi.Insecure,
	}

	esxiClient, err := client.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	ops := vm.NewOperations(esxiClient)
	ctx := context.Background()

	vmObj, err := esxiClient.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	snapshots, err := ops.ListSnapshots(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	snapshotRef, found := findSnapshotByName(snapshots, snapshotName)
	if !found {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	fmt.Printf("Reverting VM '%s' to snapshot '%s'...\n", vmName, snapshotName)

	err = ops.RevertToSnapshot(ctx, vmObj, snapshotRef)
	if err != nil {
		return fmt.Errorf("failed to revert to snapshot: %w", err)
	}

	fmt.Printf("VM '%s' reverted to snapshot '%s' successfully\n", vmName, snapshotName)
	return nil
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	snapshotName := args[1]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	clientConfig := &client.Config{
		Host:     cfg.ESXi.Host,
		User:     cfg.ESXi.User,
		Password: cfg.ESXi.Password,
		Insecure: cfg.ESXi.Insecure,
	}

	esxiClient, err := client.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create ESXi client: %w", err)
	}
	defer esxiClient.Close()

	ops := vm.NewOperations(esxiClient)
	ctx := context.Background()

	vmObj, err := esxiClient.FindVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	snapshots, err := ops.ListSnapshots(ctx, vmObj)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	snapshotRef, found := findSnapshotByName(snapshots, snapshotName)
	if !found {
		return fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	fmt.Printf("Deleting snapshot '%s' from VM '%s'...\n", snapshotName, vmName)
	if snapshotFlags.children {
		fmt.Println("Will also delete child snapshots")
	}

	err = ops.RemoveSnapshot(ctx, vmObj, snapshotRef, snapshotFlags.children)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	fmt.Printf("Snapshot '%s' deleted successfully\n", snapshotName)
	return nil
}

func findSnapshotByName(snapshots []types.VirtualMachineSnapshotTree, name string) (types.ManagedObjectReference, bool) {
	for _, snap := range snapshots {
		if snap.Name == name {
			return snap.Snapshot, true
		}
		if len(snap.ChildSnapshotList) > 0 {
			if ref, found := findSnapshotByName(snap.ChildSnapshotList, name); found {
				return ref, true
			}
		}
	}
	return types.ManagedObjectReference{}, false
}