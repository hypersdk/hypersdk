// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/apimachinery/pkg/runtime/schema"

	hypersdk "hypersdk/pkg/apis/hypersdk/v1alpha1"
)

// handleVMCreate creates a new VM manifest
func handleVMCreate(kubeconfig, namespace, name string, cpus int, memory, image, template, output string, interactive bool, gpuCount int, gpuVendor, gpuModel string, gpuPassthrough bool) {
	// If interactive mode, prompt for all parameters
	if interactive {
		pterm.DefaultHeader.WithFullWidth().Println("Interactive VM Creation Wizard")
		pterm.Println()

		// Prompt for VM name
		if name == "" {
			namePrompt := &survey.Input{
				Message: "VM Name:",
				Help:    "Enter a unique name for your virtual machine",
			}
			survey.AskOne(namePrompt, &name, survey.WithValidator(survey.Required))
		}

		// Prompt for namespace
		if namespace == "" || namespace == "default" {
			nsPrompt := &survey.Input{
				Message: "Namespace:",
				Default: "default",
				Help:    "Kubernetes namespace for the VM",
			}
			survey.AskOne(nsPrompt, &namespace)
		}

		// Prompt for CPUs
		if cpus == 2 {
			cpuInput := ""
			cpuPrompt := &survey.Input{
				Message: "Number of CPUs:",
				Default: "2",
				Help:    "Number of virtual CPU cores (e.g., 2, 4, 8)",
			}
			survey.AskOne(cpuPrompt, &cpuInput)
			if cpuInput != "" {
				if parsed, err := strconv.Atoi(cpuInput); err == nil {
					cpus = parsed
				}
			}
		}

		// Prompt for memory
		if memory == "4Gi" {
			memPrompt := &survey.Input{
				Message: "Memory:",
				Default: "4Gi",
				Help:    "Memory size (e.g., 2Gi, 4Gi, 8Gi, 16Gi)",
			}
			survey.AskOne(memPrompt, &memory)
		}

		// Prompt for GPUs
		if gpuCount == 0 {
			gpuInput := ""
			gpuPrompt := &survey.Input{
				Message: "Number of GPUs:",
				Default: "0",
				Help:    "Number of GPUs to attach (0 for no GPU, 1-8 for GPU count)",
			}
			survey.AskOne(gpuPrompt, &gpuInput)
			if gpuInput != "" {
				if parsed, err := strconv.Atoi(gpuInput); err == nil {
					gpuCount = parsed
				}
			}
		}

		// If GPUs are requested, prompt for vendor and model
		if gpuCount > 0 {
			vendorPrompt := &survey.Select{
				Message: "GPU Vendor:",
				Options: []string{"nvidia", "amd", "intel"},
				Default: "nvidia",
				Help:    "Select GPU vendor",
			}
			survey.AskOne(vendorPrompt, &gpuVendor)

			if gpuModel == "" {
				modelPrompt := &survey.Input{
					Message: "GPU Model (optional):",
					Help:    "GPU model name (e.g., Tesla V100, RTX 3090)",
				}
				survey.AskOne(modelPrompt, &gpuModel)
			}

			passthroughPrompt := &survey.Confirm{
				Message: "Enable full GPU passthrough?",
				Default: true,
				Help:    "Full passthrough gives the VM exclusive access to the GPU",
			}
			survey.AskOne(passthroughPrompt, &gpuPassthrough)
		}

		// Prompt for image source or template
		sourceType := ""
		sourcePrompt := &survey.Select{
			Message: "VM Source:",
			Options: []string{"Container Image", "VM Template", "None (blank VM)"},
			Default: "Container Image",
			Help:    "Choose how to create the VM",
		}
		survey.AskOne(sourcePrompt, &sourceType)

		switch sourceType {
		case "Container Image":
			if image == "" {
				imagePrompt := &survey.Input{
					Message: "Image Source:",
					Default: "ubuntu:22.04",
					Help:    "Container image or disk image URL",
				}
				survey.AskOne(imagePrompt, &image)
			}
			template = "" // Clear template if set
		case "VM Template":
			if template == "" {
				templatePrompt := &survey.Input{
					Message: "Template Name:",
					Help:    "Name of existing VMTemplate resource",
				}
				survey.AskOne(templatePrompt, &template)
			}
			image = "" // Clear image if set
		case "None (blank VM)":
			image = ""
			template = ""
		}

		// Confirm before creating
		confirm := false
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Create VM '%s' with %d CPUs and %s memory?", name, cpus, memory),
			Default: true,
		}
		survey.AskOne(confirmPrompt, &confirm)

		if !confirm {
			pterm.Warning.Println("VM creation cancelled")
			os.Exit(0)
		}

		pterm.Success.Printf("Creating VM: %s\n", name)
		pterm.Println()
	}

	// Validate required parameters
	if name == "" {
		fmt.Println("Error: VM name is required (-vm)")
		os.Exit(1)
	}

	if namespace == "" {
		namespace = "default"
	}

	// Build VM spec
	vm := &hypersdk.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VirtualMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hypersdk.VirtualMachineSpec{
			CPUs:    int32(cpus),
			Memory:  memory,
			Running: true,
		},
	}

	// Add image or template
	if template != "" {
		vm.Spec.Image = &hypersdk.VMImage{
			TemplateRef: &hypersdk.TemplateRef{
				Name: template,
			},
		}
	} else if image != "" {
		vm.Spec.Image = &hypersdk.VMImage{
			Source: image,
		}
	}

	// Add default disk
	bootOrder := int32(1)
	vm.Spec.Disks = []hypersdk.VMDisk{
		{
			Name:         "root",
			Size:         "20Gi",
			StorageClass: "standard",
			BootOrder:    &bootOrder,
		},
	}

	// Add GPUs if requested
	if gpuCount > 0 {
		// Determine resource name based on vendor
		resourceName := fmt.Sprintf("%s.com/gpu", gpuVendor)
		if gpuVendor == "nvidia" {
			resourceName = "nvidia.com/gpu"
		} else if gpuVendor == "amd" {
			resourceName = "amd.com/gpu"
		} else if gpuVendor == "intel" {
			resourceName = "gpu.intel.com/i915"
		}

		vm.Spec.GPUs = []hypersdk.VMGPU{
			{
				Name:         fmt.Sprintf("gpu-%s", gpuVendor),
				DeviceName:   resourceName,
				Vendor:       gpuVendor,
				Model:        gpuModel,
				Count:        int32(gpuCount),
				Passthrough:  gpuPassthrough,
				VirtualGPU:   !gpuPassthrough,
				ResourceName: resourceName,
			},
		}

		pterm.Info.Printf("Added %d %s GPU(s) to VM spec\n", gpuCount, gpuVendor)
	}

	// Add default network
	vm.Spec.Networks = []hypersdk.VMNetwork{
		{
			Name: "default",
			Type: "pod-network",
		},
	}

	// Output VM manifest
	outputResource(vm, output)
}

// waitForVMOperation waits for a VMOperation to complete with optional progress bar
func waitForVMOperation(kubeconfig, namespace, operationName string, showProgress bool, timeoutSeconds int) error {
	// Get kubeconfig path
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Build config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("error building kubeconfig: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Define VMOperation GVR
	vmOpGVR := schema.GroupVersionResource{
		Group:    "hypersdk.io",
		Version:  "v1alpha1",
		Resource: "vmoperations",
	}

	client := dynamicClient.Resource(vmOpGVR).Namespace(namespace)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	var progressBar *pterm.ProgressbarPrinter
	if showProgress {
		progressBar, _ = pterm.DefaultProgressbar.WithTotal(100).WithTitle("Operation Progress").Start()
		defer progressBar.Stop()
	}

	// Poll for operation status
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			if showProgress {
				progressBar.Stop()
			}
			return fmt.Errorf("operation timed out after %d seconds", timeoutSeconds)
		case <-ticker.C:
			// Get VMOperation status
			vmOp, err := client.Get(context.Background(), operationName, metav1.GetOptions{})
			if err != nil {
				if showProgress {
					progressBar.Stop()
				}
				return fmt.Errorf("error getting operation status: %w", err)
			}

			// Extract status
			status, ok := vmOp.Object["status"].(map[string]interface{})
			if !ok {
				continue
			}

			phase, _ := status["phase"].(string)
			progress, _ := status["progress"].(float64)
			message, _ := status["message"].(string)

			// Update progress bar
			if showProgress {
				progressBar.UpdateTitle(fmt.Sprintf("Operation: %s", phase))
				if progress > 0 {
					progressBar.Current = int(progress)
				}
				if message != "" {
					pterm.Info.Println(message)
				}
			} else {
				// Simple status update
				elapsed := time.Since(startTime).Round(time.Second)
				fmt.Printf("[%s] Phase: %s, Progress: %.0f%%, Message: %s\n", elapsed, phase, progress, message)
			}

			// Check if completed
			switch phase {
			case "Succeeded":
				if showProgress {
					progressBar.Current = 100
					progressBar.Stop()
					pterm.Success.Printf("Operation completed successfully in %s\n", time.Since(startTime).Round(time.Second))
				} else {
					fmt.Printf("✓ Operation completed successfully in %s\n", time.Since(startTime).Round(time.Second))
				}
				return nil
			case "Failed":
				if showProgress {
					progressBar.Stop()
				}
				errorMsg, _ := status["error"].(string)
				return fmt.Errorf("operation failed: %s", errorMsg)
			case "Cancelled":
				if showProgress {
					progressBar.Stop()
				}
				return fmt.Errorf("operation was cancelled")
			}
		}
	}
}

// waitForVMSnapshot waits for a VMSnapshot to be ready
func waitForVMSnapshot(kubeconfig, namespace, snapshotName string, showProgress bool, timeoutSeconds int) error {
	// Get kubeconfig path
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Build config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("error building kubeconfig: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Define VMSnapshot GVR
	vmSnapshotGVR := schema.GroupVersionResource{
		Group:    "hypersdk.io",
		Version:  "v1alpha1",
		Resource: "vmsnapshots",
	}

	client := dynamicClient.Resource(vmSnapshotGVR).Namespace(namespace)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	var progressBar *pterm.ProgressbarPrinter
	if showProgress {
		progressBar, _ = pterm.DefaultProgressbar.WithTotal(100).WithTitle("Snapshot Progress").Start()
		defer progressBar.Stop()
	}

	// Poll for snapshot status
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			if showProgress {
				progressBar.Stop()
			}
			return fmt.Errorf("snapshot creation timed out after %d seconds", timeoutSeconds)
		case <-ticker.C:
			// Get VMSnapshot status
			vmSnapshot, err := client.Get(context.Background(), snapshotName, metav1.GetOptions{})
			if err != nil {
				if showProgress {
					progressBar.Stop()
				}
				return fmt.Errorf("error getting snapshot status: %w", err)
			}

			// Extract status
			status, ok := vmSnapshot.Object["status"].(map[string]interface{})
			if !ok {
				continue
			}

			phase, _ := status["phase"].(string)
			ready, _ := status["ready"].(bool)

			// Update progress bar (simulate progress since snapshots don't report percentage)
			if showProgress {
				elapsed := time.Since(startTime).Seconds()
				estimatedProgress := int(elapsed / float64(timeoutSeconds) * 100)
				if estimatedProgress > 90 {
					estimatedProgress = 90
				}
				progressBar.Current = estimatedProgress
				progressBar.UpdateTitle(fmt.Sprintf("Snapshot: %s", phase))
			} else {
				// Simple status update
				elapsed := time.Since(startTime).Round(time.Second)
				fmt.Printf("[%s] Phase: %s, Ready: %v\n", elapsed, phase, ready)
			}

			// Check if ready
			if ready {
				if showProgress {
					progressBar.Current = 100
					progressBar.Stop()
					pterm.Success.Printf("Snapshot created successfully in %s\n", time.Since(startTime).Round(time.Second))
				} else {
					fmt.Printf("✓ Snapshot created successfully in %s\n", time.Since(startTime).Round(time.Second))
				}
				return nil
			}

			// Check for failure
			if phase == "Failed" {
				if showProgress {
					progressBar.Stop()
				}
				errorMsg, _ := status["error"].(string)
				return fmt.Errorf("snapshot creation failed: %s", errorMsg)
			}
		}
	}
}

// handleVMList lists VMs
func handleVMList(kubeconfig, namespace string, allNamespaces bool, output string, watchMode bool,
	filterStatus, filterNode, filterLabels string, filterMinCPUs int, filterMinMemory string) {
	// Get kubeconfig path
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Build config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Define VM GVR
	vmGVR := schema.GroupVersionResource{
		Group:    "hypersdk.io",
		Version:  "v1alpha1",
		Resource: "virtualmachines",
	}

	// Set namespace for query
	var client dynamic.ResourceInterface
	if allNamespaces {
		client = dynamicClient.Resource(vmGVR)
	} else {
		client = dynamicClient.Resource(vmGVR).Namespace(namespace)
	}

	ctx := context.Background()

	if watchMode {
		// Watch mode - continuous updates
		watcher, err := client.Watch(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error watching VMs: %v\n", err)
			os.Exit(1)
		}
		defer watcher.Stop()

		fmt.Println("Watching for VM changes... (Press Ctrl+C to stop)")
		fmt.Println()

		for event := range watcher.ResultChan() {
			vm := event.Object
			timestamp := time.Now().Format("15:04:05")

			// Get VM name and phase
			metadata, _ := vm.(*metav1.PartialObjectMetadata)
			if metadata != nil {
				eventType := string(event.Type)
				fmt.Printf("[%s] %s: %s/%s\n", timestamp, eventType, metadata.Namespace, metadata.Name)
			}
		}
	} else {
		// List mode - one-time query
		listOpts := metav1.ListOptions{}

		// Apply label selector if specified
		if filterLabels != "" {
			listOpts.LabelSelector = filterLabels
		}

		list, err := client.List(ctx, listOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing VMs: %v\n", err)
			os.Exit(1)
		}

		// Apply additional filters (status, node, resource requirements)
		filteredItems := list.Items
		if filterStatus != "" || filterNode != "" || filterMinCPUs > 0 || filterMinMemory != "" {
			filteredItems = filterVMs(list.Items, filterStatus, filterNode, filterMinCPUs, filterMinMemory)
		}

		// Create filtered list for output
		filteredList := list.DeepCopy()
		filteredList.Items = filteredItems

		// Output based on format
		switch output {
		case "json":
			data, _ := json.MarshalIndent(filteredList, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(filteredList)
			fmt.Println(string(data))
		default:
			// Table format
			fmt.Printf("%-30s %-15s %-15s %-10s\n", "NAME", "NAMESPACE", "PHASE", "NODE")
			fmt.Println("─────────────────────────────────────────────────────────────────────────")

			if len(filteredItems) == 0 {
				fmt.Println("No virtual machines found matching criteria")
			} else {
				fmt.Printf("Found %d virtual machine(s)\n", len(filteredItems))
				fmt.Println()
				fmt.Println("Use kubectl to view details:")
				if allNamespaces {
					fmt.Println("  kubectl get vm --all-namespaces")
				} else {
					fmt.Printf("  kubectl get vm -n %s\n", namespace)
				}
			}
		}
	}
}

// handleVMGet gets VM details
func handleVMGet(kubeconfig, namespace, name, output string, watchMode bool) {
	// Get kubeconfig path
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Build config
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Define VM GVR
	vmGVR := schema.GroupVersionResource{
		Group:    "hypersdk.io",
		Version:  "v1alpha1",
		Resource: "virtualmachines",
	}

	client := dynamicClient.Resource(vmGVR).Namespace(namespace)
	ctx := context.Background()

	if watchMode {
		// Watch mode - monitor changes to specific VM
		watcher, err := client.Watch(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error watching VM: %v\n", err)
			os.Exit(1)
		}
		defer watcher.Stop()

		fmt.Printf("Watching VM %s/%s... (Press Ctrl+C to stop)\n", namespace, name)
		fmt.Println()

		for event := range watcher.ResultChan() {
			timestamp := time.Now().Format("15:04:05")

			switch event.Type {
			case watch.Added:
				fmt.Printf("[%s] VM ADDED\n", timestamp)
			case watch.Modified:
				fmt.Printf("[%s] VM MODIFIED\n", timestamp)
			case watch.Deleted:
				fmt.Printf("[%s] VM DELETED\n", timestamp)
			case watch.Error:
				fmt.Printf("[%s] ERROR\n", timestamp)
			}

			// Print VM details based on output format
			vm := event.Object
			switch output {
			case "json":
				data, _ := json.MarshalIndent(vm, "", "  ")
				fmt.Println(string(data))
			case "yaml":
				data, _ := yaml.Marshal(vm)
				fmt.Println(string(data))
			default:
				// Simple status output
				fmt.Printf("Event: %s\n", event.Type)
			}
			fmt.Println("─────────────────────────────────────────────────────────────────────────")
		}
	} else {
		// Get mode - one-time query
		vm, err := client.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting VM: %v\n", err)
			os.Exit(1)
		}

		// Output based on format
		switch output {
		case "json":
			data, _ := json.MarshalIndent(vm, "", "  ")
			fmt.Println(string(data))
		case "yaml":
			data, _ := yaml.Marshal(vm)
			fmt.Println(string(data))
		default:
			fmt.Printf("VM: %s/%s\n", namespace, name)
			fmt.Println()
			fmt.Println("Use kubectl to view full details:")
			fmt.Printf("  kubectl get vm %s -n %s -o yaml\n", name, namespace)
		}
	}
}

// handleVMDelete deletes a VM
func handleVMDelete(kubeconfig, namespace, name string, force bool) {
	fmt.Println("VM delete functionality - to be implemented")
	fmt.Printf("Use: kubectl delete vm %s -n %s\n", name, namespace)
}

// handleVMDescribe describes a VM
func handleVMDescribe(kubeconfig, namespace, name string) {
	fmt.Println("VM describe functionality - to be implemented")
	fmt.Printf("Use: kubectl describe vm %s -n %s\n", name, namespace)
}

// handleVMStart starts a VM
func handleVMStart(kubeconfig, namespace, name, output string) {
	op := &hypersdk.VMOperation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMOperation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-start", name),
			Namespace: namespace,
		},
		Spec: hypersdk.VMOperationSpec{
			VMRef: hypersdk.VMReference{
				Name:      name,
				Namespace: namespace,
			},
			Operation: hypersdk.VMOpStart,
		},
	}
	outputResource(op, output)
}

// handleVMStop stops a VM
func handleVMStop(kubeconfig, namespace, name, output string) {
	op := &hypersdk.VMOperation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMOperation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-stop", name),
			Namespace: namespace,
		},
		Spec: hypersdk.VMOperationSpec{
			VMRef: hypersdk.VMReference{
				Name:      name,
				Namespace: namespace,
			},
			Operation: hypersdk.VMOpStop,
		},
	}
	outputResource(op, output)
}

// handleVMRestart restarts a VM
func handleVMRestart(kubeconfig, namespace, name, output string) {
	op := &hypersdk.VMOperation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMOperation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-restart", name),
			Namespace: namespace,
		},
		Spec: hypersdk.VMOperationSpec{
			VMRef: hypersdk.VMReference{
				Name:      name,
				Namespace: namespace,
			},
			Operation: hypersdk.VMOpRestart,
		},
	}
	outputResource(op, output)
}

// handleVMClone clones a VM
func handleVMClone(kubeconfig, namespace, source, target, output string, wait, showProgress bool, timeout int) {
	linked := false
	opName := fmt.Sprintf("%s-clone-%d", source, time.Now().Unix())
	op := &hypersdk.VMOperation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMOperation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opName,
			Namespace: namespace,
		},
		Spec: hypersdk.VMOperationSpec{
			VMRef: hypersdk.VMReference{
				Name:      source,
				Namespace: namespace,
			},
			Operation: hypersdk.VMOpClone,
			CloneSpec: &hypersdk.CloneSpec{
				TargetName:  target,
				LinkedClone: linked,
			},
		},
	}

	outputResource(op, output)

	if wait {
		fmt.Println()
		pterm.Info.Printf("Waiting for clone operation to complete...\n")
		fmt.Println()

		if err := waitForVMOperation(kubeconfig, namespace, opName, showProgress, timeout); err != nil {
			pterm.Error.Printf("Failed to wait for operation: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleVMMigrate migrates a VM
func handleVMMigrate(kubeconfig, namespace, name, targetNode, output string, wait, showProgress bool, timeout int) {
	opName := fmt.Sprintf("%s-migrate-%d", name, time.Now().Unix())
	op := &hypersdk.VMOperation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMOperation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opName,
			Namespace: namespace,
		},
		Spec: hypersdk.VMOperationSpec{
			VMRef: hypersdk.VMReference{
				Name:      name,
				Namespace: namespace,
			},
			Operation: hypersdk.VMOpMigrate,
			MigrateSpec: &hypersdk.MigrateSpec{
				TargetNode: targetNode,
				Live:       true,
			},
		},
	}

	outputResource(op, output)

	if wait {
		fmt.Println()
		pterm.Info.Printf("Waiting for migration to complete...\n")
		fmt.Println()

		if err := waitForVMOperation(kubeconfig, namespace, opName, showProgress, timeout); err != nil {
			pterm.Error.Printf("Failed to wait for operation: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleVMResize resizes a VM
func handleVMResize(kubeconfig, namespace, name string, cpus int, memory, output string, wait, showProgress bool, timeout int) {
	opName := fmt.Sprintf("%s-resize-%d", name, time.Now().Unix())
	op := &hypersdk.VMOperation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMOperation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opName,
			Namespace: namespace,
		},
		Spec: hypersdk.VMOperationSpec{
			VMRef: hypersdk.VMReference{
				Name:      name,
				Namespace: namespace,
			},
			Operation: hypersdk.VMOpResize,
			ResizeSpec: &hypersdk.ResizeSpec{
				CPUs:   int32(cpus),
				Memory: memory,
			},
		},
	}

	outputResource(op, output)

	if wait {
		fmt.Println()
		pterm.Info.Printf("Waiting for resize operation to complete...\n")
		fmt.Println()

		if err := waitForVMOperation(kubeconfig, namespace, opName, showProgress, timeout); err != nil {
			pterm.Error.Printf("Failed to wait for operation: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleVMSnapshotCreate creates a VM snapshot
func handleVMSnapshotCreate(kubeconfig, namespace, vmName, snapshotName string, includeMemory bool, output string, wait, showProgress bool, timeout int) {
	snapshot := &hypersdk.VMSnapshot{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMSnapshot",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshotName,
			Namespace: namespace,
		},
		Spec: hypersdk.VMSnapshotSpec{
			VMRef: hypersdk.VMReference{
				Name:      vmName,
				Namespace: namespace,
			},
			IncludeMemory: includeMemory,
		},
	}

	outputResource(snapshot, output)

	if wait {
		fmt.Println()
		pterm.Info.Printf("Waiting for snapshot creation to complete...\n")
		fmt.Println()

		if err := waitForVMSnapshot(kubeconfig, namespace, snapshotName, showProgress, timeout); err != nil {
			pterm.Error.Printf("Failed to wait for snapshot: %v\n", err)
			os.Exit(1)
		}
	}
}

// handleVMSnapshotList lists VM snapshots
func handleVMSnapshotList(kubeconfig, namespace string, allNamespaces bool, output string) {
	fmt.Println("VM snapshot list - to be implemented")
	fmt.Printf("Use: kubectl get vmsnapshots -n %s\n", namespace)
}

// handleVMSnapshotDelete deletes a VM snapshot
func handleVMSnapshotDelete(kubeconfig, namespace, name string, force bool) {
	fmt.Println("VM snapshot delete - to be implemented")
	fmt.Printf("Use: kubectl delete vmsnapshot %s -n %s\n", name, namespace)
}

// handleTemplateList lists VM templates
func handleTemplateList(kubeconfig, namespace string, allNamespaces bool, output string) {
	fmt.Println("VM template list - to be implemented")
	fmt.Printf("Use: kubectl get vmtemplates -n %s\n", namespace)
}

// handleTemplateGet gets a VM template
func handleTemplateGet(kubeconfig, namespace, name, output string) {
	fmt.Println("VM template get - to be implemented")
	fmt.Printf("Use: kubectl get vmtemplate %s -n %s -o yaml\n", name, namespace)
}

// outputResource outputs a resource in YAML or JSON format
func outputResource(resource interface{}, format string) {
	if format == "json" {
		data, err := json.MarshalIndent(resource, "", "  ")
		if err != nil {
			fmt.Printf("Error: Failed to marshal to JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		// Default to YAML
		data, err := yaml.Marshal(resource)
		if err != nil {
			fmt.Printf("Error: Failed to marshal to YAML: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	}
}

// filterVMs filters VM list based on criteria
func filterVMs(items []unstructured.Unstructured, filterStatus, filterNode string, filterMinCPUs int, filterMinMemory string) []unstructured.Unstructured {
	filtered := make([]unstructured.Unstructured, 0)

	for _, item := range items {
		// Get the object as map
		vmMap := item.Object

		// Check status filter
		if filterStatus != "" {
			status, ok := vmMap["status"].(map[string]interface{})
			if ok {
				phase, _ := status["phase"].(string)
				// Match phase (case-insensitive)
				if !matchesPhase(phase, filterStatus) {
					continue
				}
			} else {
				// No status, skip if filtering by status
				continue
			}
		}

		// Check node filter
		if filterNode != "" {
			status, ok := vmMap["status"].(map[string]interface{})
			if ok {
				nodeName, _ := status["nodeName"].(string)
				if nodeName != filterNode {
					continue
				}
			} else {
				continue
			}
		}

		// Check CPU filter
		if filterMinCPUs > 0 {
			spec, ok := vmMap["spec"].(map[string]interface{})
			if ok {
				cpus, _ := spec["cpus"].(float64) // JSON numbers are float64
				if int(cpus) < filterMinCPUs {
					continue
				}
			} else {
				continue
			}
		}

		// Check memory filter (simplified - just check if specified memory contains the value)
		if filterMinMemory != "" {
			spec, ok := vmMap["spec"].(map[string]interface{})
			if ok {
				memory, _ := spec["memory"].(string)
				// Simple string comparison (in production, parse Gi/Mi/etc)
				if memory < filterMinMemory {
					continue
				}
			} else {
				continue
			}
		}

		// VM passed all filters
		filtered = append(filtered, item)
	}

	return filtered
}

// matchesPhase checks if phase matches the filter (case-insensitive, partial match)
func matchesPhase(phase, filter string) bool {
	// Normalize to lowercase for comparison
	phaseLower := fmt.Sprintf("%v", phase)
	filterLower := fmt.Sprintf("%v", filter)

	// Support multiple statuses separated by comma
	filters := []string{filterLower}
	if len(filterLower) > 0 && filterLower[0] != 0 {
		// Simple case: exact match or contains
		for _, f := range filters {
			if len(f) > 0 && (phaseLower == f || phaseLower == f+"ning" || phaseLower == f+"ped") {
				return true
			}
			// Check if phase starts with filter
			if len(phaseLower) >= len(f) && phaseLower[:len(f)] == f {
				return true
			}
		}
	}
	return false
}

// handleVMCloneFromSnapshot clones a VM from a snapshot
func handleVMCloneFromSnapshot(kubeconfig, namespace, snapshotName, targetVM, output string, wait, showProgress bool, timeout int) {
	// Create a VMOperation for cloning from snapshot
	opName := fmt.Sprintf("%s-clone-from-snapshot-%d", targetVM, time.Now().Unix())
	
	op := &hypersdk.VMOperation{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMOperation",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opName,
			Namespace: namespace,
		},
		Spec: hypersdk.VMOperationSpec{
			VMRef: hypersdk.VMReference{
				Name:      targetVM,
				Namespace: namespace,
			},
			Operation: hypersdk.VMOpClone,
			CloneSpec: &hypersdk.CloneSpec{
				TargetName:   targetVM,
				SnapshotRef:  snapshotName,
				LinkedClone:  false,
				PowerOnAfter: true,
			},
		},
	}

	outputResource(op, output)

	if wait {
		fmt.Println()
		pterm.Info.Printf("Creating VM from snapshot %s...\n", snapshotName)
		fmt.Println()

		if err := waitForVMOperation(kubeconfig, namespace, opName, showProgress, timeout); err != nil {
			pterm.Error.Printf("Failed to wait for operation: %v\n", err)
			os.Exit(1)
		}

		pterm.Success.Printf("Successfully created VM %s from snapshot %s\n", targetVM, snapshotName)
	}
}
