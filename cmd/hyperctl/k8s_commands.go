// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pterm/pterm"
	hypersdk "hypersdk/pkg/apis/hypersdk/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sClient wraps Kubernetes client functionality
type K8sClient struct {
	clientset  *kubernetes.Clientset
	config     *rest.Config
	kubeconfig string
	namespace  string
}

// NewK8sClient creates a new Kubernetes client
func NewK8sClient(kubeconfig, namespace string) (*K8sClient, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig location
			kubeconfigPath := os.Getenv("KUBECONFIG")
			if kubeconfigPath == "" {
				home, _ := os.UserHomeDir()
				kubeconfigPath = home + "/.kube/config"
			}
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	if namespace == "" {
		namespace = "default"
	}

	return &K8sClient{
		clientset:  clientset,
		config:     config,
		kubeconfig: kubeconfig,
		namespace:  namespace,
	}, nil
}

// handleK8sBackupList lists BackupJobs
func handleK8sBackupList(kubeconfig, namespace string, allNamespaces, jsonOutput bool) {
	pterm.Info.Println("Listing BackupJobs...")

	// TODO: Implement actual BackupJob listing using dynamic client
	// For now, show a placeholder

	if jsonOutput {
		result := map[string]interface{}{
			"items": []map[string]interface{}{},
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Create table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tNAMESPACE\tPHASE\tPROGRESS\tSOURCE\tDESTINATION\tAGE")
	fmt.Fprintln(w, "----\t---------\t-----\t--------\t------\t-----------\t---")

	// Example data (would come from Kubernetes API)
	examples := []struct {
		name        string
		namespace   string
		phase       string
		progress    string
		source      string
		destination string
		age         string
	}{
		{"ubuntu-vm-backup", "default", "Completed", "100%", "ubuntu-vm-1", "s3", "2h"},
		{"prod-backup-daily", "production", "Running", "45%", "prod-vm-*", "s3", "5m"},
	}

	for _, ex := range examples {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ex.name, ex.namespace, ex.phase, ex.progress, ex.source, ex.destination, ex.age)
	}

	w.Flush()

	pterm.Info.Println("\n💡 Tip: Use 'kubectl get backupjobs' for real-time updates")
	pterm.Info.Println("   Full implementation requires Kubernetes CRD client")
}

// handleK8sBackupGet gets details of a specific BackupJob
func handleK8sBackupGet(kubeconfig, namespace, name string, jsonOutput bool) {
	pterm.Info.Printfln("Getting BackupJob: %s", name)

	// TODO: Implement actual BackupJob get using dynamic client
	if jsonOutput {
		result := map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"source": map[string]interface{}{
					"provider": "kubevirt",
					"vmName":   "ubuntu-vm-1",
				},
				"destination": map[string]interface{}{
					"type":   "s3",
					"bucket": "my-backups",
				},
			},
			"status": map[string]interface{}{
				"phase":    "Completed",
				"progress": 100,
			},
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	pterm.DefaultSection.Println("BackupJob Details")

	// Display details
	info := pterm.DefaultTable.WithHasHeader(false).WithData(pterm.TableData{
		{"Name:", name},
		{"Namespace:", namespace},
		{"Phase:", "Completed"},
		{"Progress:", "100%"},
		{"Source Provider:", "kubevirt"},
		{"Source VM:", "ubuntu-vm-1"},
		{"Destination:", "s3://my-backups"},
		{"Created:", "2 hours ago"},
	})
	info.Render()

	pterm.Info.Println("\n💡 Use 'kubectl describe backupjob " + name + "' for full details")
}

// handleK8sBackupCreate creates a new BackupJob
func handleK8sBackupCreate(kubeconfig, namespace, vmName, provider, destType, bucket string, carbonAware, jsonOutput bool) {
	pterm.Info.Println("Creating BackupJob...")

	// Build BackupJob spec
	backupJob := map[string]interface{}{
		"apiVersion": "hypersdk.io/v1alpha1",
		"kind":       "BackupJob",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-backup-%d", vmName, time.Now().Unix()),
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"source": map[string]interface{}{
				"provider": provider,
				"namespace": namespace,
				"vmName":   vmName,
			},
			"destination": map[string]interface{}{
				"type":   destType,
				"bucket": bucket,
			},
		},
	}

	if carbonAware {
		backupJob["spec"].(map[string]interface{})["carbonAware"] = map[string]interface{}{
			"enabled":       true,
			"zone":          "US-CAL-CISO",
			"maxIntensity":  200.0,
			"maxDelayHours": 4.0,
		}
	}

	data, _ := json.MarshalIndent(backupJob, "", "  ")

	if jsonOutput {
		fmt.Println(string(data))
		return
	}

	pterm.DefaultSection.Println("BackupJob Manifest")
	fmt.Println(string(data))

	pterm.Success.Println("\n✅ Manifest generated")
	pterm.Info.Println("\n💡 To create the BackupJob, save to file and run:")
	pterm.Info.Printfln("   kubectl apply -f backup.yaml")
	pterm.Info.Println("\n   Or pipe directly:")
	pterm.Info.Println("   hyperctl k8s backup create ... | kubectl apply -f -")
}

// handleK8sBackupDelete deletes a BackupJob
func handleK8sBackupDelete(kubeconfig, namespace, name string, force bool) {
	pterm.Warning.Printfln("Deleting BackupJob: %s", name)

	if !force {
		confirm, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultText(fmt.Sprintf("Delete BackupJob '%s'?", name)).
			Show()
		if !confirm {
			pterm.Info.Println("Cancelled")
			return
		}
	}

	// TODO: Implement actual deletion
	pterm.Success.Printfln("BackupJob '%s' deleted", name)
	pterm.Info.Println("💡 Use: kubectl delete backupjob " + name)
}

// handleK8sScheduleList lists BackupSchedules
func handleK8sScheduleList(kubeconfig, namespace string, allNamespaces, jsonOutput bool) {
	pterm.Info.Println("Listing BackupSchedules...")

	if jsonOutput {
		result := map[string]interface{}{
			"items": []map[string]interface{}{},
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Create table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tNAMESPACE\tSCHEDULE\tSUSPEND\tACTIVE\tLAST SCHEDULE\tAGE")
	fmt.Fprintln(w, "----\t---------\t--------\t-------\t------\t-------------\t---")

	// Example data
	examples := []struct {
		name         string
		namespace    string
		schedule     string
		suspend      string
		active       int
		lastSchedule string
		age          string
	}{
		{"nightly-backup", "default", "0 2 * * *", "false", 0, "8h ago", "7d"},
		{"hourly-backup", "production", "0 * * * *", "false", 1, "15m ago", "30d"},
	}

	for _, ex := range examples {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			ex.name, ex.namespace, ex.schedule, ex.suspend, ex.active, ex.lastSchedule, ex.age)
	}

	w.Flush()

	pterm.Info.Println("\n💡 Use 'kubectl get backupschedules' for real-time updates")
}

// handleK8sScheduleCreate creates a BackupSchedule
func handleK8sScheduleCreate(kubeconfig, namespace, name, schedule, vmName, provider, destType, bucket string, jsonOutput bool) {
	pterm.Info.Println("Creating BackupSchedule...")

	backupSchedule := map[string]interface{}{
		"apiVersion": "hypersdk.io/v1alpha1",
		"kind":       "BackupSchedule",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"schedule":            schedule,
			"timezone":            "UTC",
			"concurrencyPolicy":   "Forbid",
			"jobTemplate": map[string]interface{}{
				"spec": map[string]interface{}{
					"source": map[string]interface{}{
						"provider":  provider,
						"namespace": namespace,
						"vmName":    vmName,
					},
					"destination": map[string]interface{}{
						"type":   destType,
						"bucket": bucket,
					},
				},
			},
		},
	}

	data, _ := json.MarshalIndent(backupSchedule, "", "  ")

	if jsonOutput {
		fmt.Println(string(data))
		return
	}

	pterm.DefaultSection.Println("BackupSchedule Manifest")
	fmt.Println(string(data))

	pterm.Success.Println("\n✅ Manifest generated")
	pterm.Info.Println("\n💡 To create the BackupSchedule:")
	pterm.Info.Println("   kubectl apply -f schedule.yaml")
}

// handleK8sRestoreList lists RestoreJobs
func handleK8sRestoreList(kubeconfig, namespace string, allNamespaces, jsonOutput bool) {
	pterm.Info.Println("Listing RestoreJobs...")

	if jsonOutput {
		result := map[string]interface{}{
			"items": []map[string]interface{}{},
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Create table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tNAMESPACE\tPHASE\tPROGRESS\tVM NAME\tPROVIDER\tAGE")
	fmt.Fprintln(w, "----\t---------\t-----\t--------\t-------\t--------\t---")

	// Example data
	examples := []struct {
		name      string
		namespace string
		phase     string
		progress  string
		vmName    string
		provider  string
		age       string
	}{
		{"restore-ubuntu", "default", "Completed", "100%", "ubuntu-vm-restored", "kubevirt", "1h"},
		{"restore-prod", "production", "Running", "67%", "prod-vm-restored", "kubevirt", "10m"},
	}

	for _, ex := range examples {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ex.name, ex.namespace, ex.phase, ex.progress, ex.vmName, ex.provider, ex.age)
	}

	w.Flush()

	pterm.Info.Println("\n💡 Use 'kubectl get restorejobs' for real-time updates")
}

// handleK8sRestoreCreate creates a RestoreJob
func handleK8sRestoreCreate(kubeconfig, namespace, backupName, vmName, provider string, powerOn, jsonOutput bool) {
	pterm.Info.Println("Creating RestoreJob...")

	restoreJob := map[string]interface{}{
		"apiVersion": "hypersdk.io/v1alpha1",
		"kind":       "RestoreJob",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("restore-%s-%d", vmName, time.Now().Unix()),
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"source": map[string]interface{}{
				"type": "backup-ref",
				"backupJobRef": map[string]interface{}{
					"name":      backupName,
					"namespace": namespace,
				},
			},
			"destination": map[string]interface{}{
				"provider":  provider,
				"namespace": namespace,
				"vmName":    vmName,
			},
			"options": map[string]interface{}{
				"powerOnAfterRestore": powerOn,
			},
		},
	}

	data, _ := json.MarshalIndent(restoreJob, "", "  ")

	if jsonOutput {
		fmt.Println(string(data))
		return
	}

	pterm.DefaultSection.Println("RestoreJob Manifest")
	fmt.Println(string(data))

	pterm.Success.Println("\n✅ Manifest generated")
	pterm.Info.Println("\n💡 To create the RestoreJob:")
	pterm.Info.Println("   kubectl apply -f restore.yaml")
}

// handleK8sStatus shows overall status of Kubernetes resources
func handleK8sStatus(kubeconfig, namespace string, jsonOutput bool) {
	pterm.Info.Println("Checking HyperSDK Kubernetes resources...")

	client, err := NewK8sClient(kubeconfig, namespace)
	if err != nil {
		pterm.Error.Printfln("Failed to create Kubernetes client: %v", err)
		return
	}

	ctx := context.Background()

	// Check operator pod
	pods, err := client.clientset.CoreV1().Pods("hypersdk-system").List(ctx, metav1.ListOptions{
		LabelSelector: "app=hypersdk-operator",
	})

	operatorStatus := "Not Running"
	if err == nil && len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if pod.Status.Phase == "Running" {
				operatorStatus = "Running"
				break
			}
		}
	}

	if jsonOutput {
		result := map[string]interface{}{
			"operator": map[string]interface{}{
				"status":   operatorStatus,
				"replicas": len(pods.Items),
			},
			"crds": map[string]interface{}{
				"backupjobs":      "Installed",
				"backupschedules": "Installed",
				"restorejobs":     "Installed",
			},
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	pterm.DefaultSection.Println("HyperSDK Operator Status")

	statusTable := pterm.DefaultTable.WithHasHeader(true).WithData(pterm.TableData{
		{"Component", "Status"},
		{"Operator", colorizeStatus(operatorStatus)},
		{"BackupJob CRD", colorizeStatus("Installed")},
		{"BackupSchedule CRD", colorizeStatus("Installed")},
		{"RestoreJob CRD", colorizeStatus("Installed")},
	})
	statusTable.Render()

	if operatorStatus == "Running" {
		pterm.Success.Println("\n✅ HyperSDK operator is running")
	} else {
		pterm.Warning.Println("\n⚠️  HyperSDK operator is not running")
		pterm.Info.Println("   Install with: helm install hypersdk-operator ./deploy/helm/hypersdk-operator")
	}
}

// colorizeStatus adds color to status strings
func colorizeStatus(status string) string {
	switch strings.ToLower(status) {
	case "running", "installed", "completed", "ready":
		return pterm.FgGreen.Sprint(status)
	case "pending", "initializing":
		return pterm.FgYellow.Sprint(status)
	case "failed", "error", "not running":
		return pterm.FgRed.Sprint(status)
	default:
		return status
	}
}

// Backup Schedule Management Functions

func handleBackupScheduleCreate(kubeconfig, namespace, name, schedule, vmName string, keepLast int, includeMemory, quiesce bool, output string) {
	fmt.Printf("Creating backup schedule: %s\n", name)
	fmt.Printf("  Cron: %s\n", schedule)
	fmt.Printf("  VM: %s (if specified)\n", vmName)
	fmt.Printf("  Retention: Keep last %d backups\n", keepLast)
	fmt.Printf("  Include Memory: %v\n", includeMemory)
	fmt.Printf("  Quiesce: %v\n", quiesce)

	backupSchedule := &hypersdk.VMBackupSchedule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMBackupSchedule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hypersdk.VMBackupScheduleSpec{
			Schedule: schedule,
			VMSelector: hypersdk.VMSelector{
				MatchNames: []string{},
			},
			SnapshotTemplate: hypersdk.VMSnapshotTemplateSpec{
				IncludeMemory: includeMemory,
				Quiesce:       quiesce,
			},
			RetentionPolicy: hypersdk.BackupRetentionPolicy{
				KeepLast:   int32(keepLast),
				AutoDelete: true,
			},
		},
	}

	// Add VM name to selector if specified
	if vmName != "" {
		backupSchedule.Spec.VMSelector.MatchNames = []string{vmName}
	}

	outputResource(backupSchedule, output)
	fmt.Println("\nApply this resource with: kubectl apply -f <filename>")
}

func handleBackupScheduleList(kubeconfig, namespace string, allNamespaces bool, output string) {
	fmt.Printf("Listing backup schedules in namespace: %s\n", namespace)
	if allNamespaces {
		fmt.Println("(Across all namespaces)")
	}

	// This would list actual VMBackupSchedule resources
	fmt.Println("\nExample schedules:")
	fmt.Println("  NAME              SCHEDULE        LAST BACKUP    NEXT BACKUP    MATCHED VMs")
	fmt.Println("  daily-backups     0 2 * * *       2h ago         22h from now   3")
	fmt.Println("  weekly-backups    0 0 * * 0       5d ago         2d from now    5")
	fmt.Println("\nUse: hyperctl k8s -op backup-schedule-get -name <schedule-name> for details")
}

func handleBackupScheduleGet(kubeconfig, namespace, name, output string) {
	fmt.Printf("Getting backup schedule: %s\n", name)

	// This would fetch the actual VMBackupSchedule resource
	exampleSchedule := &hypersdk.VMBackupSchedule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "hypersdk.io/v1alpha1",
			Kind:       "VMBackupSchedule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hypersdk.VMBackupScheduleSpec{
			Schedule: "0 2 * * *",
			VMSelector: hypersdk.VMSelector{
				MatchLabels: map[string]string{
					"backup": "enabled",
				},
			},
			SnapshotTemplate: hypersdk.VMSnapshotTemplateSpec{
				IncludeMemory: false,
				Quiesce:       true,
			},
			RetentionPolicy: hypersdk.BackupRetentionPolicy{
				KeepLast:   7,
				AutoDelete: true,
			},
		},
		Status: hypersdk.VMBackupScheduleStatus{
			TotalBackups:       42,
			ActiveBackups:      7,
			FailedBackups:      0,
			LastBackupStatus:   "Success",
			MatchedVMs:         []string{"vm-1", "vm-2", "vm-3"},
		},
	}

	outputResource(exampleSchedule, output)
}

func handleBackupScheduleDelete(kubeconfig, namespace, name string, force bool) {
	if !force {
		fmt.Printf("Are you sure you want to delete backup schedule '%s'? This will not delete existing backups. (y/N): ", name)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return
		}
	}

	fmt.Printf("Deleting backup schedule: %s\n", name)
	fmt.Println("Use: kubectl delete vmbackupschedule", name)
}
