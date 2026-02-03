// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"

	"hypersdk/cmd/completion"
	"hypersdk/daemon/models"
	"hypersdk/providers/vsphere"
)

const (
	defaultDaemonURL = "http://localhost:8080"
	version          = "0.0.1"
)

var (
	funFacts = []string{
		"Tip: Use -json flag for automation and scripting",
		"Tip: Increase parallel_downloads for faster exports",
		"Tip: Monitor jobs with: hyperctl query -status running",
		"Tip: Remove CD/DVD devices before migration",
		"Tip: Use batch files for exporting multiple VMs",
	}

	motivationalMessages = []string{
		"Operation completed successfully",
		"Task finished",
		"Export job submitted",
		"Configuration updated",
		"Command executed",
	}
)

// matchResult represents a VM search match result
type matchResult struct {
	vm            vsphere.VMInfo
	matchCount    int
	matchedFields []string
}

func showBanner() {
	// Orange/amber color scheme (Claude-inspired)
	orange := pterm.NewStyle(pterm.FgLightRed)
	amber := pterm.NewStyle(pterm.FgYellow)

	banner := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("HYPER", orange),
		pterm.NewLettersFromStringWithStyle("CTL", amber),
	)
	banner.Render()

	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		pterm.LightYellow("Multi-cloud VM migration control CLI\n") +
			pterm.Gray("Version "+version),
	)
}

func showRandomFact() {
	// Note: rand is auto-seeded in Go 1.20+, no manual seeding needed
	fact := funFacts[rand.Intn(len(funFacts))]
	pterm.Println()
	pterm.FgLightCyan.Println(fact)
}

func showMotivation() {
	// Note: rand is auto-seeded in Go 1.20+, no manual seeding needed
	msg := motivationalMessages[rand.Intn(len(motivationalMessages))]
	pterm.Println()
	pterm.FgLightGreen.Println(msg)
}

func showSuccessArt() {
	pterm.Println()
	pterm.DefaultCenter.Println(pterm.LightGreen("✨ ⭐ 🎉 SUCCESS! 🎉 ⭐ ✨"))
	pterm.Println()
}

func main() {
	// Global flags
	daemonURL := flag.String("daemon", defaultDaemonURL, "Daemon URL")
	versionFlag := flag.Bool("version", false, "Show version")

	// Define subcommands
	submitCmd := flag.NewFlagSet("submit", flag.ExitOnError)
	submitFile := submitCmd.String("file", "", "Job file (JSON/YAML)")
	submitVM := submitCmd.String("vm", "", "VM path")
	submitOutput := submitCmd.String("output", "", "Output directory")

	queryCmd := flag.NewFlagSet("query", flag.ExitOnError)
	queryAll := queryCmd.Bool("all", false, "Query all jobs")
	queryID := queryCmd.String("id", "", "Query specific job ID")
	queryStatus := queryCmd.String("status", "", "Filter by status (running,completed,failed)")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listJSON := listCmd.Bool("json", false, "Output in JSON format")
	listFilter := listCmd.String("filter", "", "Filter VMs by name (case-insensitive)")

	grepCmd := flag.NewFlagSet("grep", flag.ExitOnError)
	grepRegex := grepCmd.Bool("E", false, "Use extended regular expressions")
	grepIgnoreCase := grepCmd.Bool("i", false, "Case-insensitive matching")
	grepInvert := grepCmd.Bool("v", false, "Invert match (show non-matching)")
	grepCount := grepCmd.Bool("c", false, "Count matching VMs")
	grepNamesOnly := grepCmd.Bool("l", false, "List VM names only")
	grepField := grepCmd.String("f", "name", "Field to search (name, path, os, power, all)")
	grepJSON := grepCmd.Bool("json", false, "Output in JSON format")

	// ripgrep-style command (rg)
	rgCmd := flag.NewFlagSet("rg", flag.ExitOnError)
	rgSmartCase := rgCmd.Bool("S", false, "Smart case (case-insensitive if pattern is lowercase)")
	rgIgnoreCase := rgCmd.Bool("i", false, "Case-insensitive matching")
	rgInvert := rgCmd.Bool("v", false, "Invert match (show non-matching)")
	rgCount := rgCmd.Bool("c", false, "Count matching VMs")
	rgCountMatches := rgCmd.Bool("count-matches", false, "Count individual matches (not just VMs)")
	rgNamesOnly := rgCmd.Bool("l", false, "List VM names only")
	rgFilesWithoutMatch := rgCmd.Bool("files-without-match", false, "List VMs that don't match")
	rgColor := rgCmd.String("color", "auto", "Colorize output (auto, always, never)")
	rgStats := rgCmd.Bool("stats", false, "Show search statistics")
	rgMultiline := rgCmd.Bool("U", false, "Enable multiline matching")
	rgMaxCount := rgCmd.Int("m", 0, "Stop after N matches (0 = unlimited)")
	rgJSON := rgCmd.Bool("json", false, "Output in JSON format")

	vmCmd := flag.NewFlagSet("vm", flag.ExitOnError)
	vmOperation := vmCmd.String("op", "", "Operation: shutdown, poweroff, remove-cdrom, info")
	vmPath := vmCmd.String("path", "", "VM path (e.g. /data/vm/my-vm)")
	vmTimeout := vmCmd.Int("timeout", 300, "Timeout in seconds (for shutdown)")

	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)

	cancelCmd := flag.NewFlagSet("cancel", flag.ExitOnError)
	cancelID := cancelCmd.String("id", "", "Job ID to cancel (comma-separated for multiple)")

	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)
	migrateOutput := migrateCmd.String("output", "/tmp/vm-migrations", "Output directory for exports")
	migrateConvert := migrateCmd.Bool("convert", true, "Auto-convert VMDK to qcow2")
	migrateImport := migrateCmd.Bool("import", false, "Auto-import to libvirt")

	completionCmd := flag.NewFlagSet("completion", flag.ExitOnError)
	completionShell := completionCmd.String("shell", "", "Shell type: bash, zsh, fish")

	watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)

	logsCmd := flag.NewFlagSet("logs", flag.ExitOnError)
	logsFollow := logsCmd.Bool("f", false, "Follow log output")
	logsTail := logsCmd.Int("tail", 50, "Number of lines to show from the end")

	schedulesCmd := flag.NewFlagSet("schedules", flag.ExitOnError)

	webhooksCmd := flag.NewFlagSet("webhooks", flag.ExitOnError)

	manifestCmd := flag.NewFlagSet("manifest", flag.ExitOnError)
	manifestAction := manifestCmd.String("action", "", "Action: create, validate, submit, generate")
	_ = manifestCmd.String("file", "", "Manifest file path") // Defined for help text, actual value from Args()

	// Workflow Commands
	workflowCmd := flag.NewFlagSet("workflow", flag.ExitOnError)
	workflowOperation := workflowCmd.String("op", "status", "Operation: status, list, queue, watch")
	workflowDir := workflowCmd.String("dir", "", "Workflow directory path")

	// vSphere Infrastructure Commands
	hostCmd := flag.NewFlagSet("host", flag.ExitOnError)
	hostOperation := hostCmd.String("op", "list", "Operation: list, info")
	hostPattern := hostCmd.String("pattern", "*", "Host name pattern")
	hostJSON := hostCmd.Bool("json", false, "JSON output")

	clusterCmd := flag.NewFlagSet("cluster", flag.ExitOnError)
	clusterOperation := clusterCmd.String("op", "list", "Operation: list, info")
	clusterPattern := clusterCmd.String("pattern", "*", "Cluster name pattern")
	clusterJSON := clusterCmd.Bool("json", false, "JSON output")

	// Performance Metrics Command
	metricsCmd := flag.NewFlagSet("metrics", flag.ExitOnError)
	metricsEntity := metricsCmd.String("entity", "", "Entity name (VM/host/cluster)")
	metricsType := metricsCmd.String("type", "vm", "Entity type: vm, host, cluster")
	metricsRealtime := metricsCmd.Bool("realtime", false, "Show realtime metrics")
	metricsStart := metricsCmd.String("start", "", "Start time (for historical)")
	metricsEnd := metricsCmd.String("end", "", "End time")
	metricsInterval := metricsCmd.String("interval", "5min", "Interval: 5min, 30min, 2hour")
	metricsWatch := metricsCmd.Bool("watch", false, "Watch metrics continuously")
	metricsJSON := metricsCmd.Bool("json", false, "JSON output")

	// Resource Pool Commands
	poolCmd := flag.NewFlagSet("pool", flag.ExitOnError)
	poolOperation := poolCmd.String("op", "list", "Operation: list, create, update, delete")
	poolName := poolCmd.String("name", "", "Pool name")
	poolParent := poolCmd.String("parent", "", "Parent pool/cluster")
	poolCPUReserve := poolCmd.Int64("cpu-reserve", 0, "CPU reservation (MHz)")
	poolCPULimit := poolCmd.Int64("cpu-limit", -1, "CPU limit (MHz, -1=unlimited)")
	poolMemReserve := poolCmd.Int64("mem-reserve", 0, "Memory reservation (MB)")
	poolMemLimit := poolCmd.Int64("mem-limit", -1, "Memory limit (MB, -1=unlimited)")
	poolJSON := poolCmd.Bool("json", false, "JSON output")

	// Events Command
	eventsCmd := flag.NewFlagSet("events", flag.ExitOnError)
	eventsSince := eventsCmd.String("since", "1h", "Show events since (duration: 1h, 24h, 7d)")
	eventsTypes := eventsCmd.String("types", "", "Filter event types (comma-separated)")
	eventsFollow := eventsCmd.Bool("follow", false, "Follow events (stream)")
	eventsJSON := eventsCmd.Bool("json", false, "JSON output")

	// Clone Command
	cloneCmd := flag.NewFlagSet("clone", flag.ExitOnError)
	cloneSource := cloneCmd.String("source", "", "Source VM name/path")
	cloneTarget := cloneCmd.String("target", "", "Target VM name")
	cloneFolder := cloneCmd.String("folder", "", "Target folder")
	clonePool := cloneCmd.String("pool", "", "Resource pool")
	cloneDatastore := cloneCmd.String("datastore", "", "Target datastore")
	clonePowerOn := cloneCmd.Bool("power-on", false, "Power on after clone")
	cloneLinked := cloneCmd.Bool("linked", false, "Create linked clone")
	cloneSnapshot := cloneCmd.String("snapshot", "", "Snapshot name (for linked clone)")
	cloneTemplate := cloneCmd.Bool("template", false, "Mark as template")
	cloneBulkFile := cloneCmd.String("bulk", "", "Bulk clone from file (YAML/JSON)")

	// Daemon Commands
	daemonCmd := flag.NewFlagSet("daemon", flag.ExitOnError)
	daemonOperation := daemonCmd.String("op", "status", "Operation: status, list")
	daemonInstance := daemonCmd.String("instance", "", "Daemon instance name (for status)")
	daemonJSON := daemonCmd.Bool("json", false, "JSON output")

	// Carbon Commands
	carbonCmd := flag.NewFlagSet("carbon", flag.ExitOnError)
	carbonOperation := carbonCmd.String("op", "status", "Operation: status, report, zones, estimate")
	carbonZone := carbonCmd.String("zone", "US-CAL-CISO", "Carbon zone (e.g., US-CAL-CISO, SE)")
	carbonThreshold := carbonCmd.Float64("threshold", 200.0, "Carbon intensity threshold (gCO2/kWh)")
	carbonJobID := carbonCmd.String("job", "", "Job ID (for report)")
	carbonStartTime := carbonCmd.String("start", "", "Start time (for report, RFC3339 format)")
	carbonEndTime := carbonCmd.String("end", "", "End time (for report, RFC3339 format)")
	carbonDataSize := carbonCmd.Float64("data", 0, "Data size in GB (for report/estimate)")
	carbonDuration := carbonCmd.Float64("hours", 2.0, "Duration in hours (for estimate)")
	carbonJSON := carbonCmd.Bool("json", false, "JSON output")

	// Kubernetes Commands
	k8sCmd := flag.NewFlagSet("k8s", flag.ExitOnError)
	k8sOperation := k8sCmd.String("op", "status", "Operation: backup-list, backup-get, backup-create, backup-delete, schedule-list, schedule-create, restore-list, restore-create, vm-create, vm-list, vm-get, vm-delete, vm-start, vm-stop, vm-clone, vm-snapshot-create, template-list, status")
	k8sKubeconfig := k8sCmd.String("kubeconfig", "", "Path to kubeconfig file")
	k8sNamespace := k8sCmd.String("namespace", "default", "Kubernetes namespace")
	k8sName := k8sCmd.String("name", "", "Resource name")
	k8sAllNamespaces := k8sCmd.Bool("all-namespaces", false, "List resources across all namespaces")
	k8sJSON := k8sCmd.Bool("json", false, "JSON output")
	k8sVMName := k8sCmd.String("vm", "", "VM name")
	k8sProvider := k8sCmd.String("provider", "kubevirt", "Provider (kubevirt, vsphere, aws, azure, gcp)")
	k8sDestType := k8sCmd.String("dest-type", "s3", "Destination type (s3, azure, gcs, local, nfs)")
	k8sBucket := k8sCmd.String("bucket", "", "Destination bucket")
	k8sCarbonAware := k8sCmd.Bool("carbon-aware", false, "Enable carbon-aware scheduling")
	k8sForce := k8sCmd.Bool("force", false, "Force operation without confirmation")
	k8sSchedule := k8sCmd.String("schedule", "", "Cron schedule (e.g., '0 2 * * *')")
	k8sBackupName := k8sCmd.String("backup", "", "BackupJob name (for restore)")
	k8sPowerOn := k8sCmd.Bool("power-on", false, "Power on after restore")
	// VM-specific flags
	k8sCPUs := k8sCmd.Int("cpus", 2, "Number of CPUs")
	k8sMemory := k8sCmd.String("memory", "4Gi", "Memory size (e.g., 4Gi, 8Gi)")
	k8sImage := k8sCmd.String("image", "", "VM image source")
	k8sTemplate := k8sCmd.String("template", "", "VM template name")
	k8sTargetVM := k8sCmd.String("target", "", "Target VM name (for clone)")
	k8sTargetNode := k8sCmd.String("target-node", "", "Target node (for migrate)")
	k8sSnapshotName := k8sCmd.String("snapshot", "", "Snapshot name")
	k8sIncludeMemory := k8sCmd.Bool("include-memory", false, "Include memory in snapshot")
	k8sOutput := k8sCmd.String("output", "yaml", "Output format (yaml, json)")
	k8sWatch := k8sCmd.Bool("watch", false, "Watch for changes in real-time")
	// Filtering options
	k8sFilterStatus := k8sCmd.String("status", "", "Filter by status (running, stopped, failed)")
	k8sFilterNode := k8sCmd.String("node", "", "Filter by node name")
	k8sFilterLabels := k8sCmd.String("selector", "", "Label selector (e.g., app=web)")
	k8sFilterMinCPUs := k8sCmd.Int("min-cpus", 0, "Minimum CPUs")
	k8sFilterMinMemory := k8sCmd.String("min-memory", "", "Minimum memory (e.g., 4Gi)")
	// Progress and wait options
	k8sWait := k8sCmd.Bool("wait", false, "Wait for operation to complete")
	k8sShowProgress := k8sCmd.Bool("show-progress", false, "Show progress bar (requires --wait)")
	k8sTimeout := k8sCmd.Int("timeout", 300, "Operation timeout in seconds (default: 300)")
	// Interactive mode
	k8sInteractive := k8sCmd.Bool("interactive", false, "Interactive mode with prompts")

	// Parse global flags
	flag.Parse()

	if *versionFlag {
		fmt.Printf("hyperctl version %s\n", version)
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	// Route to subcommands
	switch os.Args[1] {
	case "submit":
		submitCmd.Parse(os.Args[2:])
		handleSubmit(*daemonURL, *submitFile, *submitVM, *submitOutput)

	case "query":
		queryCmd.Parse(os.Args[2:])
		handleQuery(*daemonURL, *queryAll, *queryID, *queryStatus)

	case "list":
		listCmd.Parse(os.Args[2:])
		handleList(*daemonURL, *listJSON, *listFilter)

	case "grep":
		grepCmd.Parse(os.Args[2:])
		if len(grepCmd.Args()) < 1 {
			pterm.Error.Println("Pattern required. Usage: hyperctl grep [OPTIONS] PATTERN")
			os.Exit(1)
		}
		pattern := grepCmd.Args()[0]
		handleGrep(*daemonURL, pattern, *grepRegex, *grepIgnoreCase, *grepInvert, *grepCount, *grepNamesOnly, *grepField, *grepJSON)

	case "rg":
		rgCmd.Parse(os.Args[2:])
		if len(rgCmd.Args()) < 1 {
			pterm.Error.Println("Pattern required. Usage: hyperctl rg [OPTIONS] PATTERN [PATTERN2 ...]")
			os.Exit(1)
		}
		patterns := rgCmd.Args()
		handleRipgrep(*daemonURL, patterns, *rgSmartCase, *rgIgnoreCase, *rgInvert, *rgCount, *rgCountMatches,
			*rgNamesOnly, *rgFilesWithoutMatch, *rgColor, *rgStats, *rgMultiline, *rgMaxCount, *rgJSON)

	case "vm":
		vmCmd.Parse(os.Args[2:])
		handleVM(*daemonURL, *vmOperation, *vmPath, *vmTimeout)

	case "status":
		statusCmd.Parse(os.Args[2:])
		handleStatus(*daemonURL)

	case "cancel":
		cancelCmd.Parse(os.Args[2:])
		handleCancel(*daemonURL, *cancelID)

	case "migrate", "interactive":
		migrateCmd.Parse(os.Args[2:])
		runInteractive(*daemonURL, *migrateOutput, *migrateConvert, *migrateImport)

	case "completion":
		completionCmd.Parse(os.Args[2:])
		handleCompletion(*completionShell)

	case "watch":
		watchCmd.Parse(os.Args[2:])
		if len(watchCmd.Args()) < 1 {
			pterm.Error.Println("Job ID required. Usage: hyperctl watch <job-id>")
			os.Exit(1)
		}
		handleWatch(*daemonURL, watchCmd.Args()[0])

	case "logs":
		logsCmd.Parse(os.Args[2:])
		if len(logsCmd.Args()) < 1 {
			pterm.Error.Println("Job ID required. Usage: hyperctl logs [-f] <job-id>")
			os.Exit(1)
		}
		handleLogs(*daemonURL, logsCmd.Args()[0], *logsFollow, *logsTail)

	case "schedules":
		schedulesCmd.Parse(os.Args[2:])
		if len(schedulesCmd.Args()) < 1 {
			pterm.Error.Println("Action required. Usage: hyperctl schedules <list|create|delete|enable|disable|trigger>")
			pterm.Info.Println("Examples:")
			pterm.Println("  hyperctl schedules list")
			pterm.Println("  hyperctl schedules create daily '0 2 * * *' -vm /dc/vm/prod -output /backups")
			pterm.Println("  hyperctl schedules enable schedule-123")
			os.Exit(1)
		}
		handleSchedules(*daemonURL, schedulesCmd.Args()[0], schedulesCmd.Args()[1:])

	case "webhooks":
		webhooksCmd.Parse(os.Args[2:])
		if len(webhooksCmd.Args()) < 1 {
			pterm.Error.Println("Action required. Usage: hyperctl webhooks <list|add|delete|test>")
			pterm.Info.Println("Examples:")
			pterm.Println("  hyperctl webhooks list")
			pterm.Println("  hyperctl webhooks add https://hooks.slack.com/xxx slack job.completed")
			pterm.Println("  hyperctl webhooks test 0")
			os.Exit(1)
		}
		handleWebhooks(*daemonURL, webhooksCmd.Args()[0], webhooksCmd.Args()[1:])

	case "manifest":
		manifestCmd.Parse(os.Args[2:])
		if *manifestAction == "" && len(manifestCmd.Args()) < 1 {
			pterm.Error.Println("Action required. Usage: hyperctl manifest <action>")
			pterm.Info.Println("Actions:")
			pterm.Println("  create     - Create new manifest interactively")
			pterm.Println("  validate   - Validate manifest file")
			pterm.Println("  submit     - Submit manifest to workflow daemon")
			pterm.Println("  generate   - Generate manifest from VM path")
			pterm.Info.Println("")
			pterm.Info.Println("Examples:")
			pterm.Println("  hyperctl manifest create")
			pterm.Println("  hyperctl manifest validate -file my-vm.json")
			pterm.Println("  hyperctl manifest submit -file my-vm.json")
			pterm.Println("  hyperctl manifest generate /dc/vm/web01 /exports")
			os.Exit(1)
		}
		action := *manifestAction
		if action == "" {
			action = manifestCmd.Args()[0]
		}
		handleManifestCmd(*daemonURL, action, manifestCmd.Args()[1:])

	case "workflow":
		workflowCmd.Parse(os.Args[2:])
		handleWorkflow(*daemonURL, *workflowOperation, *workflowDir)

	case "host":
		hostCmd.Parse(os.Args[2:])
		handleHost(*daemonURL, *hostOperation, *hostPattern, *hostJSON)

	case "cluster":
		clusterCmd.Parse(os.Args[2:])
		handleCluster(*daemonURL, *clusterOperation, *clusterPattern, *clusterJSON)

	case "metrics":
		metricsCmd.Parse(os.Args[2:])
		if *metricsEntity == "" {
			pterm.Error.Println("Entity name required (-entity)")
			os.Exit(1)
		}
		handleMetrics(*daemonURL, *metricsEntity, *metricsType, *metricsRealtime,
			*metricsStart, *metricsEnd, *metricsInterval, *metricsWatch, *metricsJSON)

	case "pool":
		poolCmd.Parse(os.Args[2:])
		handlePool(*daemonURL, *poolOperation, *poolName, *poolParent,
			*poolCPUReserve, *poolCPULimit, *poolMemReserve, *poolMemLimit, *poolJSON)

	case "events":
		eventsCmd.Parse(os.Args[2:])
		types := []string{}
		if *eventsTypes != "" {
			types = strings.Split(*eventsTypes, ",")
		}
		handleEvents(*daemonURL, *eventsSince, types, *eventsFollow, *eventsJSON)

	case "clone":
		cloneCmd.Parse(os.Args[2:])
		if *cloneBulkFile != "" {
			handleClone(*daemonURL, vsphere.CloneSpec{}, *cloneBulkFile)
		} else {
			if *cloneSource == "" || *cloneTarget == "" {
				pterm.Error.Println("Source and target required (-source, -target)")
				os.Exit(1)
			}
			spec := vsphere.CloneSpec{
				SourceVM:     *cloneSource,
				TargetName:   *cloneTarget,
				TargetFolder: *cloneFolder,
				ResourcePool: *clonePool,
				Datastore:    *cloneDatastore,
				PowerOn:      *clonePowerOn,
				LinkedClone:  *cloneLinked,
				Snapshot:     *cloneSnapshot,
				Template:     *cloneTemplate,
			}
			handleClone(*daemonURL, spec, "")
		}

	case "daemon":
		daemonCmd.Parse(os.Args[2:])
		switch *daemonOperation {
		case "status":
			handleDaemonStatus(*daemonURL, *daemonInstance, *daemonJSON)
		case "list":
			handleDaemonList(*daemonJSON)
		default:
			pterm.Error.Printfln("Unknown daemon operation: %s", *daemonOperation)
			pterm.Info.Println("Available operations: status, list")
			os.Exit(1)
		}

	case "carbon":
		carbonCmd.Parse(os.Args[2:])
		switch *carbonOperation {
		case "status":
			handleCarbonStatus(*daemonURL, *carbonZone, *carbonThreshold, *carbonJSON)
		case "report":
			if *carbonJobID == "" || *carbonDataSize == 0 || *carbonStartTime == "" || *carbonEndTime == "" {
				pterm.Error.Println("For report: -job, -data, -start, and -end are required")
				pterm.Info.Println("Example: hyperctl carbon -op report -job job-123 -data 500 -start 2026-02-04T10:00:00Z -end 2026-02-04T12:00:00Z -zone US-CAL-CISO")
				os.Exit(1)
			}
			start, err := parseTime(*carbonStartTime)
			if err != nil {
				pterm.Error.Printfln("Invalid start time: %v", err)
				os.Exit(1)
			}
			end, err := parseTime(*carbonEndTime)
			if err != nil {
				pterm.Error.Printfln("Invalid end time: %v", err)
				os.Exit(1)
			}
			handleCarbonReport(*daemonURL, *carbonJobID, *carbonZone, start, end, *carbonDataSize, *carbonJSON)
		case "zones":
			handleCarbonZones(*daemonURL, *carbonJSON)
		case "estimate":
			if *carbonDataSize == 0 {
				pterm.Error.Println("For estimate: -data is required")
				pterm.Info.Println("Example: hyperctl carbon -op estimate -zone US-CAL-CISO -data 500 -hours 2")
				os.Exit(1)
			}
			handleCarbonEstimate(*daemonURL, *carbonZone, *carbonDataSize, *carbonDuration, *carbonJSON)
		default:
			pterm.Error.Printfln("Unknown carbon operation: %s", *carbonOperation)
			pterm.Info.Println("Available operations: status, report, zones, estimate")
			pterm.Println()
			pterm.Info.Println("Examples:")
			pterm.Println("  hyperctl carbon -op status -zone US-CAL-CISO")
			pterm.Println("  hyperctl carbon -op zones")
			pterm.Println("  hyperctl carbon -op estimate -zone US-CAL-CISO -data 500 -hours 2")
			pterm.Println("  hyperctl carbon -op report -job job-123 -data 500 -start 2026-02-04T10:00:00Z -end 2026-02-04T12:00:00Z -zone US-CAL-CISO")
			os.Exit(1)
		}

	case "k8s":
		k8sCmd.Parse(os.Args[2:])
		switch *k8sOperation {
		case "backup-list":
			handleK8sBackupList(*k8sKubeconfig, *k8sNamespace, *k8sAllNamespaces, *k8sJSON)
		case "backup-get":
			if *k8sName == "" {
				pterm.Error.Println("BackupJob name required (-name)")
				os.Exit(1)
			}
			handleK8sBackupGet(*k8sKubeconfig, *k8sNamespace, *k8sName, *k8sJSON)
		case "backup-create":
			if *k8sVMName == "" || *k8sBucket == "" {
				pterm.Error.Println("VM name and bucket required (-vm, -bucket)")
				pterm.Info.Println("Example: hyperctl k8s -op backup-create -vm ubuntu-vm-1 -bucket my-backups -carbon-aware")
				os.Exit(1)
			}
			handleK8sBackupCreate(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sProvider, *k8sDestType, *k8sBucket, *k8sCarbonAware, *k8sJSON)
		case "backup-delete":
			if *k8sName == "" {
				pterm.Error.Println("BackupJob name required (-name)")
				os.Exit(1)
			}
			handleK8sBackupDelete(*k8sKubeconfig, *k8sNamespace, *k8sName, *k8sForce)
		case "schedule-list":
			handleK8sScheduleList(*k8sKubeconfig, *k8sNamespace, *k8sAllNamespaces, *k8sJSON)
		case "schedule-create":
			if *k8sName == "" || *k8sSchedule == "" || *k8sVMName == "" || *k8sBucket == "" {
				pterm.Error.Println("Name, schedule, VM name, and bucket required (-name, -schedule, -vm, -bucket)")
				pterm.Info.Println("Example: hyperctl k8s -op schedule-create -name nightly -schedule '0 2 * * *' -vm my-vm -bucket backups")
				os.Exit(1)
			}
			handleK8sScheduleCreate(*k8sKubeconfig, *k8sNamespace, *k8sName, *k8sSchedule, *k8sVMName, *k8sProvider, *k8sDestType, *k8sBucket, *k8sJSON)
		case "restore-list":
			handleK8sRestoreList(*k8sKubeconfig, *k8sNamespace, *k8sAllNamespaces, *k8sJSON)
		case "restore-create":
			if *k8sBackupName == "" || *k8sVMName == "" {
				pterm.Error.Println("Backup name and VM name required (-backup, -vm)")
				pterm.Info.Println("Example: hyperctl k8s -op restore-create -backup my-vm-backup -vm my-vm-restored -power-on")
				os.Exit(1)
			}
			handleK8sRestoreCreate(*k8sKubeconfig, *k8sNamespace, *k8sBackupName, *k8sVMName, *k8sProvider, *k8sPowerOn, *k8sJSON)
		case "status":
			handleK8sStatus(*k8sKubeconfig, *k8sNamespace, *k8sJSON)

		// VM Management Operations
		case "vm-create":
			if !*k8sInteractive && *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm) or use --interactive")
				pterm.Info.Println("Example: hyperctl k8s -op vm-create -vm my-vm -cpus 4 -memory 8Gi -image ubuntu:22.04")
				pterm.Info.Println("Or:      hyperctl k8s -op vm-create --interactive")
				os.Exit(1)
			}
			handleVMCreate(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sCPUs, *k8sMemory, *k8sImage, *k8sTemplate, *k8sOutput, *k8sInteractive)
		case "vm-list":
			handleVMList(*k8sKubeconfig, *k8sNamespace, *k8sAllNamespaces, *k8sOutput, *k8sWatch,
				*k8sFilterStatus, *k8sFilterNode, *k8sFilterLabels, *k8sFilterMinCPUs, *k8sFilterMinMemory)
		case "vm-get":
			if *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm)")
				os.Exit(1)
			}
			handleVMGet(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sOutput, *k8sWatch)
		case "vm-delete":
			if *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm)")
				os.Exit(1)
			}
			handleVMDelete(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sForce)
		case "vm-describe":
			if *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm)")
				os.Exit(1)
			}
			handleVMDescribe(*k8sKubeconfig, *k8sNamespace, *k8sVMName)
		case "vm-start":
			if *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm)")
				os.Exit(1)
			}
			handleVMStart(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sOutput)
		case "vm-stop":
			if *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm)")
				os.Exit(1)
			}
			handleVMStop(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sOutput)
		case "vm-restart":
			if *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm)")
				os.Exit(1)
			}
			handleVMRestart(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sOutput)
		case "vm-clone":
			if *k8sVMName == "" || *k8sTargetVM == "" {
				pterm.Error.Println("Source VM and target VM names required (-vm, -target)")
				pterm.Info.Println("Example: hyperctl k8s -op vm-clone -vm my-vm -target my-vm-clone --wait --show-progress")
				os.Exit(1)
			}
			handleVMClone(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sTargetVM, *k8sOutput, *k8sWait, *k8sShowProgress, *k8sTimeout)
		case "vm-migrate":
			if *k8sVMName == "" || *k8sTargetNode == "" {
				pterm.Error.Println("VM name and target node required (-vm, -target-node)")
				pterm.Info.Println("Example: hyperctl k8s -op vm-migrate -vm my-vm -target-node node-2 --wait --show-progress")
				os.Exit(1)
			}
			handleVMMigrate(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sTargetNode, *k8sOutput, *k8sWait, *k8sShowProgress, *k8sTimeout)
		case "vm-resize":
			if *k8sVMName == "" {
				pterm.Error.Println("VM name required (-vm)")
				pterm.Info.Println("Example: hyperctl k8s -op vm-resize -vm my-vm -cpus 4 -memory 8Gi --wait --show-progress")
				os.Exit(1)
			}
			handleVMResize(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sCPUs, *k8sMemory, *k8sOutput, *k8sWait, *k8sShowProgress, *k8sTimeout)
		case "vm-snapshot-create":
			if *k8sVMName == "" || *k8sSnapshotName == "" {
				pterm.Error.Println("VM name and snapshot name required (-vm, -snapshot)")
				pterm.Info.Println("Example: hyperctl k8s -op vm-snapshot-create -vm my-vm -snapshot snap1 -include-memory --wait --show-progress")
				os.Exit(1)
			}
			handleVMSnapshotCreate(*k8sKubeconfig, *k8sNamespace, *k8sVMName, *k8sSnapshotName, *k8sIncludeMemory, *k8sOutput, *k8sWait, *k8sShowProgress, *k8sTimeout)
		case "vm-snapshot-list":
			handleVMSnapshotList(*k8sKubeconfig, *k8sNamespace, *k8sAllNamespaces, *k8sOutput)
		case "vm-snapshot-delete":
			if *k8sSnapshotName == "" {
				pterm.Error.Println("Snapshot name required (-snapshot)")
				os.Exit(1)
			}
			handleVMSnapshotDelete(*k8sKubeconfig, *k8sNamespace, *k8sSnapshotName, *k8sForce)
		case "template-list":
			handleTemplateList(*k8sKubeconfig, *k8sNamespace, *k8sAllNamespaces, *k8sOutput)
		case "template-get":
			if *k8sTemplate == "" {
				pterm.Error.Println("Template name required (-template)")
				os.Exit(1)
			}
			handleTemplateGet(*k8sKubeconfig, *k8sNamespace, *k8sTemplate, *k8sOutput)

		default:
			pterm.Error.Printfln("Unknown k8s operation: %s", *k8sOperation)
			pterm.Info.Println("Available operations:")
			pterm.Println("  Backup & Restore:")
			pterm.Println("    backup-list       - List BackupJobs")
			pterm.Println("    backup-get        - Get BackupJob details")
			pterm.Println("    backup-create     - Create BackupJob manifest")
			pterm.Println("    backup-delete     - Delete BackupJob")
			pterm.Println("    schedule-list     - List BackupSchedules")
			pterm.Println("    schedule-create   - Create BackupSchedule manifest")
			pterm.Println("    restore-list      - List RestoreJobs")
			pterm.Println("    restore-create    - Create RestoreJob manifest")
			pterm.Println()
			pterm.Println("  VM Management:")
			pterm.Println("    vm-create         - Create VirtualMachine")
			pterm.Println("    vm-list           - List VirtualMachines")
			pterm.Println("    vm-get            - Get VirtualMachine details")
			pterm.Println("    vm-delete         - Delete VirtualMachine")
			pterm.Println("    vm-describe       - Describe VirtualMachine")
			pterm.Println("    vm-start          - Start VirtualMachine")
			pterm.Println("    vm-stop           - Stop VirtualMachine")
			pterm.Println("    vm-restart        - Restart VirtualMachine")
			pterm.Println("    vm-clone          - Clone VirtualMachine")
			pterm.Println("    vm-migrate        - Migrate VirtualMachine")
			pterm.Println("    vm-resize         - Resize VirtualMachine")
			pterm.Println()
			pterm.Println("  Snapshots:")
			pterm.Println("    vm-snapshot-create  - Create VM snapshot")
			pterm.Println("    vm-snapshot-list    - List VM snapshots")
			pterm.Println("    vm-snapshot-delete  - Delete VM snapshot")
			pterm.Println()
			pterm.Println("  Templates:")
			pterm.Println("    template-list     - List VM templates")
			pterm.Println("    template-get      - Get template details")
			pterm.Println()
			pterm.Println("  Other:")
			pterm.Println("    status            - Show operator status")
			pterm.Println()
			pterm.Info.Println("Examples:")
			pterm.Println("  hyperctl k8s -op status")
			pterm.Println("  hyperctl k8s -op vm-list -namespace default")
			pterm.Println("  hyperctl k8s -op vm-create --interactive")
			pterm.Println("  hyperctl k8s -op vm-create -vm ubuntu-vm-1 -cpus 4 -memory 8Gi -image ubuntu:22.04")
			pterm.Println("  hyperctl k8s -op vm-start -vm ubuntu-vm-1")
			pterm.Println("  hyperctl k8s -op vm-clone -vm my-vm -target my-vm-clone")
			pterm.Println("  hyperctl k8s -op vm-snapshot-create -vm my-vm -snapshot snap1 -include-memory")
			pterm.Println("  hyperctl k8s -op backup-create -vm ubuntu-vm-1 -bucket my-backups -carbon-aware")
			os.Exit(1)
		}

	case "help", "-h", "--help":
		showUsage()

	default:
		pterm.Error.Printfln("Unknown command: %s", os.Args[1])
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	// Show banner
	showBanner()

	pterm.Println()
	pterm.Info.Println("🚀 A powerful CLI for multi-cloud to KVM migration")
	pterm.Println()

	// VM Discovery Commands
	pterm.DefaultSection.Println("📋 VM Discovery")
	discoveryCommands := [][]string{
		{"Command", "Description", "Example"},
		{"list", "List VMs from vCenter", "hyperctl list"},
		{"list -json", "List VMs (JSON output)", "hyperctl list -json"},
		{"list -filter", "Filter VMs by name", "hyperctl list -filter rhel"},
		{"grep PATTERN", "Search VMs (grep-like)", "hyperctl grep ubuntu"},
		{"grep -i PATTERN", "Case-insensitive search", "hyperctl grep -i windows"},
		{"grep -E PATTERN", "Regex search", "hyperctl grep -E '^web-.*'"},
		{"grep -v PATTERN", "Invert match", "hyperctl grep -v test"},
		{"grep -c PATTERN", "Count matches", "hyperctl grep -c production"},
		{"grep -l PATTERN", "List names only", "hyperctl grep -l ubuntu"},
		{"grep -f field", "Search specific field", "hyperctl grep -f os centos"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(discoveryCommands).
		Render()

	pterm.Println()

	// Grep field options
	pterm.Info.Println("Grep fields: name, path, os, power, all")

	pterm.Println()

	// Ripgrep-style search
	pterm.DefaultSection.Println("🔎 Advanced Search (ripgrep)")
	rgCommands := [][]string{
		{"Command", "Description", "Example"},
		{"rg PATTERN", "Smart search (auto case-insensitive)", "hyperctl rg ubuntu"},
		{"rg -S PATTERN", "Smart case matching", "hyperctl rg -S Ubuntu"},
		{"rg -i PATTERN", "Case-insensitive", "hyperctl rg -i WINDOWS"},
		{"rg PAT1 PAT2", "Multiple patterns (OR)", "hyperctl rg web db"},
		{"rg -v PATTERN", "Invert match", "hyperctl rg -v test"},
		{"rg -c PATTERN", "Count matching VMs", "hyperctl rg -c prod"},
		{"rg --count-matches", "Count all matches", "hyperctl rg --count-matches server"},
		{"rg -l PATTERN", "List names only", "hyperctl rg -l ubuntu"},
		{"rg --stats PATTERN", "Show search statistics", "hyperctl rg --stats web"},
		{"rg -m N PATTERN", "Limit to N matches", "hyperctl rg -m 10 prod"},
		{"rg --color always", "Force colored output", "hyperctl rg --color always web"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(rgCommands).
		Render()

	pterm.Println()

	// VM Operations Commands
	pterm.DefaultSection.Println("🔧 VM Operations")
	vmCommands := [][]string{
		{"Command", "Description", "Example"},
		{"vm -op shutdown", "Graceful VM shutdown", "hyperctl vm -op shutdown -path /data/vm/my-vm"},
		{"vm -op poweroff", "Force power off VM", "hyperctl vm -op poweroff -path /data/vm/my-vm"},
		{"vm -op remove-cdrom", "Remove CD/DVD devices", "hyperctl vm -op remove-cdrom -path /data/vm/my-vm"},
		{"vm -op info", "Get VM details", "hyperctl vm -op info -path /data/vm/my-vm"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(vmCommands).
		Render()

	pterm.Println()

	// Interactive Migration
	pterm.DefaultSection.Println("🎮 Interactive Migration")
	migrateCommands := [][]string{
		{"Command", "Description", "Example"},
		{"migrate", "Interactive VM selection & migration", "hyperctl migrate"},
		{"migrate -output", "Set output directory", "hyperctl migrate -output /migrations"},
		{"migrate -convert=false", "Skip auto-conversion to qcow2", "hyperctl migrate -convert=false"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(migrateCommands).
		Render()

	pterm.Println()

	// Job Management Commands
	pterm.DefaultSection.Println("📦 Job Management")
	jobCommands := [][]string{
		{"Command", "Description", "Example"},
		{"submit", "Submit export job", "hyperctl submit -vm /data/vm/my-vm -output /tmp"},
		{"submit -file", "Submit from YAML/JSON", "hyperctl submit -file jobs.yaml"},
		{"query", "Query job status", "hyperctl query -all"},
		{"status", "Show daemon status", "hyperctl status"},
		{"cancel", "Cancel running jobs", "hyperctl cancel -id abc123"},
		{"watch <job-id>", "Watch job in real-time", "hyperctl watch abc123"},
		{"logs <job-id>", "View job logs", "hyperctl logs abc123"},
		{"logs -f <job-id>", "Follow job logs", "hyperctl logs -f abc123"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(jobCommands).
		Render()

	pterm.Println()

	// Automation & Scheduling
	pterm.DefaultSection.Println("⏰ Automation & Scheduling")
	scheduleCommands := [][]string{
		{"Command", "Description", "Example"},
		{"schedules list", "List all schedules", "hyperctl schedules list"},
		{"schedules create", "Create new schedule", "hyperctl schedules create daily '0 2 * * *' -vm /dc/vm/prod"},
		{"schedules enable <id>", "Enable a schedule", "hyperctl schedules enable schedule-123"},
		{"schedules disable <id>", "Disable a schedule", "hyperctl schedules disable schedule-123"},
		{"schedules trigger <id>", "Run schedule now", "hyperctl schedules trigger schedule-123"},
		{"schedules delete <id>", "Delete a schedule", "hyperctl schedules delete schedule-123"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(scheduleCommands).
		Render()

	pterm.Println()

	// Webhooks & Notifications
	pterm.DefaultSection.Println("🔔 Webhooks & Notifications")
	webhookCommands := [][]string{
		{"Command", "Description", "Example"},
		{"webhooks list", "List all webhooks", "hyperctl webhooks list"},
		{"webhooks add", "Add new webhook", "hyperctl webhooks add https://hooks.slack.com/xxx slack job.completed"},
		{"webhooks test <index>", "Test a webhook", "hyperctl webhooks test 0"},
		{"webhooks delete <index>", "Delete a webhook", "hyperctl webhooks delete 0"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(webhookCommands).
		Render()

	pterm.Println()

	// Workflow Management
	pterm.DefaultSection.Println("🔄 Workflow Management (NEW)")
	workflowCommands := [][]string{
		{"Command", "Description", "Example"},
		{"workflow -op status", "Show workflow daemon status", "hyperctl workflow -op status"},
		{"workflow -op list", "List workflow jobs", "hyperctl workflow -op list"},
		{"workflow -op queue", "Show queue statistics", "hyperctl workflow -op queue"},
		{"workflow -op watch", "Watch workflow directory", "hyperctl workflow -op watch -dir /path"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(workflowCommands).
		Render()

	pterm.Println()

	// Manifest Management
	pterm.DefaultSection.Println("📝 Manifest Management (NEW)")
	manifestCommands := [][]string{
		{"Command", "Description", "Example"},
		{"manifest create", "Create manifest interactively", "hyperctl manifest create"},
		{"manifest validate", "Validate manifest file", "hyperctl manifest validate -file vm.json"},
		{"manifest submit", "Submit to workflow daemon", "hyperctl manifest submit -file vm.json"},
		{"manifest generate", "Generate from VM path", "hyperctl manifest generate /dc/vm/web01 /exports"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(manifestCommands).
		Render()

	pterm.Println()

	// Daemon Management
	pterm.DefaultSection.Println("🔧 Daemon Management (hyper2kvm)")
	daemonCommands := [][]string{
		{"Command", "Description", "Example"},
		{"daemon -op status", "Show daemon status", "hyperctl daemon -op status"},
		{"daemon -op status -instance", "Show specific instance status", "hyperctl daemon -op status -instance vsphere"},
		{"daemon -op list", "List all daemon instances", "hyperctl daemon -op list"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(daemonCommands).
		Render()

	pterm.Println()

	// Carbon-Aware Scheduling
	pterm.DefaultSection.Println("🌿 Carbon-Aware Scheduling (NEW)")
	carbonCommands := [][]string{
		{"Command", "Description", "Example"},
		{"carbon -op status", "Check grid carbon status", "hyperctl carbon -op status -zone US-CAL-CISO"},
		{"carbon -op zones", "List available carbon zones", "hyperctl carbon -op zones"},
		{"carbon -op estimate", "Estimate carbon savings", "hyperctl carbon -op estimate -zone US-CAL-CISO -data 500 -hours 2"},
		{"carbon -op report", "Generate carbon report for job", "hyperctl carbon -op report -job job-123 -data 500 -start 2026-02-04T10:00:00Z -end 2026-02-04T12:00:00Z -zone US-CAL-CISO"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(carbonCommands).
		Render()

	pterm.Println()

	// Kubernetes Integration
	pterm.DefaultSection.Println("☸️  Kubernetes Integration (NEW)")
	k8sCommands := [][]string{
		{"Command", "Description", "Example"},
		{"k8s -op status", "Show operator status", "hyperctl k8s -op status"},
		{"k8s -op backup-list", "List BackupJobs", "hyperctl k8s -op backup-list -namespace default"},
		{"k8s -op backup-get", "Get BackupJob details", "hyperctl k8s -op backup-get -name my-vm-backup"},
		{"k8s -op backup-create", "Create BackupJob manifest", "hyperctl k8s -op backup-create -vm ubuntu-vm-1 -bucket my-backups"},
		{"k8s -op backup-delete", "Delete BackupJob", "hyperctl k8s -op backup-delete -name my-vm-backup"},
		{"k8s -op schedule-list", "List BackupSchedules", "hyperctl k8s -op schedule-list"},
		{"k8s -op schedule-create", "Create BackupSchedule", "hyperctl k8s -op schedule-create -name nightly -schedule '0 2 * * *' -vm my-vm -bucket backups"},
		{"k8s -op restore-list", "List RestoreJobs", "hyperctl k8s -op restore-list"},
		{"k8s -op restore-create", "Create RestoreJob", "hyperctl k8s -op restore-create -backup my-vm-backup -vm my-vm-restored -power-on"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(k8sCommands).
		Render()

	pterm.Println()

	// VM Management
	pterm.DefaultSection.Println("🖥️  VM Management (Kubernetes)")
	vmMgmtCommands := [][]string{
		{"Command", "Description", "Example"},
		{"k8s -op vm-create --interactive", "Create VM (wizard mode)", "hyperctl k8s -op vm-create --interactive"},
		{"k8s -op vm-create", "Create VirtualMachine", "hyperctl k8s -op vm-create -vm my-vm -cpus 4 -memory 8Gi"},
		{"k8s -op vm-list", "List VirtualMachines", "hyperctl k8s -op vm-list -namespace default"},
		{"k8s -op vm-get", "Get VM details", "hyperctl k8s -op vm-get -vm my-vm"},
		{"k8s -op vm-delete", "Delete VirtualMachine", "hyperctl k8s -op vm-delete -vm my-vm"},
		{"k8s -op vm-start", "Start VM", "hyperctl k8s -op vm-start -vm my-vm"},
		{"k8s -op vm-stop", "Stop VM", "hyperctl k8s -op vm-stop -vm my-vm"},
		{"k8s -op vm-clone", "Clone VM", "hyperctl k8s -op vm-clone -vm my-vm -target my-vm-2"},
		{"k8s -op vm-migrate", "Migrate VM to node", "hyperctl k8s -op vm-migrate -vm my-vm -target-node node-2"},
		{"k8s -op vm-snapshot-create", "Create VM snapshot", "hyperctl k8s -op vm-snapshot-create -vm my-vm -snapshot snap1"},
		{"k8s -op template-list", "List VM templates", "hyperctl k8s -op template-list"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(vmMgmtCommands).
		Render()

	pterm.Println()

	// Shell Completion
	pterm.DefaultSection.Println("🔧 Shell Completion")
	completionCommands := [][]string{
		{"Command", "Description", "Example"},
		{"completion -shell bash", "Generate bash completion", "hyperctl completion -shell bash"},
		{"completion -shell zsh", "Generate zsh completion", "hyperctl completion -shell zsh"},
		{"completion -shell fish", "Generate fish completion", "hyperctl completion -shell fish"},
	}
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(completionCommands).
		Render()

	pterm.Println()
	pterm.Info.Println("Examples:")
	pterm.Println("  # Interactive Migration")
	pterm.Println("  hyperctl migrate                                  # Launch interactive mode")
	pterm.Println("  hyperctl migrate -output /migrations              # Custom output directory")
	pterm.Println()
	pterm.Println("  # VM Discovery")
	pterm.Println("  hyperctl list                                     # List all VMs")
	pterm.Println("  hyperctl list -json                               # List VMs in JSON format")
	pterm.Println("  hyperctl list -filter rhel                        # Filter VMs by name")
	pterm.Println()
	pterm.Println("  # VM Operations")
	pterm.Println("  hyperctl vm -op shutdown -path /data/vm/my-vm     # Graceful shutdown")
	pterm.Println("  hyperctl vm -op poweroff -path /data/vm/my-vm     # Force power off")
	pterm.Println("  hyperctl vm -op remove-cdrom -path /data/vm/my-vm # Remove CD/DVD")
	pterm.Println("  hyperctl vm -op info -path /data/vm/my-vm         # Get VM info")
	pterm.Println()
	pterm.Println("  # Job Management")
	pterm.Println("  hyperctl submit -file jobs.yaml")
	pterm.Println("  hyperctl submit -vm /DC/vm/test-vm -output /tmp/export")
	pterm.Println("  hyperctl query -all")
	pterm.Println("  hyperctl query -id abc123")
	pterm.Println("  hyperctl query -status running")
	pterm.Println("  hyperctl status")
	pterm.Println("  hyperctl cancel -id abc123,def456")
	pterm.Println()
	pterm.Println("  # Shell Completion")
	pterm.Println("  hyperctl completion -shell bash > /etc/bash_completion.d/hyperctl")
	pterm.Println("  hyperctl completion -shell zsh > ${fpath[1]}/_hyperctl")
	pterm.Println("  hyperctl completion -shell fish > ~/.config/fish/completions/hyperctl.fish")
}

func handleSubmit(daemonURL, filePath, vmPath, outputPath string) {
	spinner, _ := pterm.DefaultSpinner.Start("Submitting job(s)...")

	var data []byte
	var contentType string
	var err error

	if filePath != "" {
		// Submit from file
		data, err = os.ReadFile(filePath)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Failed to read file: %v", err))
			os.Exit(1)
		}

		ext := filepath.Ext(filePath)
		if ext == ".yaml" || ext == ".yml" {
			contentType = "application/x-yaml"
		} else {
			contentType = "application/json"
		}
	} else if vmPath != "" {
		// Create job from command line args
		if outputPath == "" {
			outputPath = fmt.Sprintf("./export-%s", filepath.Base(vmPath))
		}

		job := models.JobDefinition{
			Name:       filepath.Base(vmPath),
			VMPath:     vmPath,
			OutputPath: outputPath,
		}

		data, err = json.Marshal(job)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Failed to create job: %v", err))
			os.Exit(1)
		}
		contentType = "application/json"
	} else {
		spinner.Fail("Either -file or -vm must be specified")
		os.Exit(1)
	}

	// Send request
	resp, err := apiRequest(daemonURL+"/jobs/submit", "POST", contentType, data)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to submit job: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	var submitResp models.SubmitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success(fmt.Sprintf("Submitted %d job(s)", submitResp.Accepted))

	// Display results
	if submitResp.Accepted > 0 {
		pterm.Success.Printfln("Accepted Jobs: %d", submitResp.Accepted)
		for _, id := range submitResp.JobIDs {
			pterm.Info.Printfln("  - Job ID: %s", id)
		}
	}

	if submitResp.Rejected > 0 {
		pterm.Warning.Printfln("Rejected Jobs: %d", submitResp.Rejected)
		for _, errMsg := range submitResp.Errors {
			pterm.Error.Printfln("  - %s", errMsg)
		}
	}
}

func handleQuery(daemonURL string, all bool, jobID, statusFilter string) {
	spinner, _ := pterm.DefaultSpinner.Start("Querying jobs...")

	req := models.QueryRequest{
		All: all,
	}

	if jobID != "" {
		req.JobIDs = strings.Split(jobID, ",")
	}

	if statusFilter != "" {
		statuses := strings.Split(statusFilter, ",")
		req.Status = make([]models.JobStatus, len(statuses))
		for i, s := range statuses {
			req.Status[i] = models.JobStatus(strings.TrimSpace(s))
		}
	}

	data, _ := json.Marshal(req)
	resp, err := apiRequest(daemonURL+"/jobs/query", "POST", "application/json", data)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to query: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	var queryResp models.QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success(fmt.Sprintf("Found %d job(s)", queryResp.Total))

	if queryResp.Total == 0 {
		pterm.Info.Println("No jobs found")
		return
	}

	// Display jobs in table
	displayJobs(queryResp.Jobs)
}

func handleList(daemonURL string, jsonOutput bool, filter string) {
	// Only show spinner if not in JSON mode
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput {
		spinner, _ = pterm.DefaultSpinner.Start("🔍 Discovering VMs from vCenter...")
	}

	// Use longer timeout for VM listing (can take time with many VMs)
	resp, err := apiRequestWithTimeout(daemonURL+"/vms/list", "GET", "", nil, 120*time.Second)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to list VMs: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		}
		os.Exit(1)
	}

	// Parse response
	var vmResp struct {
		VMs       []vsphere.VMInfo `json:"vms"`
		Total     int              `json:"total"`
		Timestamp time.Time        `json:"timestamp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Apply filter if specified
	vms := vmResp.VMs
	if filter != "" {
		var filtered []vsphere.VMInfo
		lowerFilter := strings.ToLower(filter)
		for _, vm := range vms {
			if strings.Contains(strings.ToLower(vm.Name), lowerFilter) ||
				strings.Contains(strings.ToLower(vm.Path), lowerFilter) {
				filtered = append(filtered, vm)
			}
		}
		vms = filtered
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("✅ Found %d VMs", len(vms)))
	}

	if len(vms) == 0 {
		pterm.Info.Println("No VMs found")
		return
	}

	// Output format
	if jsonOutput {
		// JSON output
		output, _ := json.MarshalIndent(map[string]interface{}{
			"vms":       vms,
			"total":     len(vms),
			"timestamp": time.Now(),
		}, "", "  ")
		fmt.Println(string(output))
	} else {
		// Display in nice table format
		displayVMs(vms)
	}
}

func handleGrep(daemonURL, pattern string, useRegex, ignoreCase, invert, count, namesOnly bool, field string, jsonOutput bool) {
	// Only show spinner if not in JSON/count/names-only mode
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput && !count && !namesOnly {
		spinner, _ = pterm.DefaultSpinner.Start("🔍 Searching VMs...")
	}

	// Fetch all VMs
	resp, err := apiRequestWithTimeout(daemonURL+"/vms/list", "GET", "", nil, 120*time.Second)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to fetch VMs: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		}
		os.Exit(1)
	}

	var vmResp struct {
		VMs       []vsphere.VMInfo `json:"vms"`
		Total     int              `json:"total"`
		Timestamp time.Time        `json:"timestamp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Compile pattern (regex or literal)
	var matcher func(string) bool
	if useRegex {
		flags := ""
		if ignoreCase {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + pattern)
		if err != nil {
			pterm.Error.Printfln("Invalid regex pattern: %v", err)
			os.Exit(1)
		}
		matcher = re.MatchString
	} else {
		// Literal string matching
		searchPattern := pattern
		if ignoreCase {
			searchPattern = strings.ToLower(pattern)
			matcher = func(s string) bool {
				return strings.Contains(strings.ToLower(s), searchPattern)
			}
		} else {
			matcher = func(s string) bool {
				return strings.Contains(s, searchPattern)
			}
		}
	}

	// Filter VMs based on field and pattern
	var matches []vsphere.VMInfo
	for _, vm := range vmResp.VMs {
		var matchText string
		switch field {
		case "name":
			matchText = vm.Name
		case "path":
			matchText = vm.Path
		case "os":
			matchText = vm.GuestOS
		case "power":
			matchText = vm.PowerState
		case "all":
			// Search across all fields
			matchText = fmt.Sprintf("%s %s %s %s", vm.Name, vm.Path, vm.GuestOS, vm.PowerState)
		default:
			pterm.Error.Printfln("Invalid field: %s. Use: name, path, os, power, all", field)
			os.Exit(1)
		}

		matched := matcher(matchText)
		if invert {
			matched = !matched
		}

		if matched {
			matches = append(matches, vm)
		}
	}

	if spinner != nil {
		spinner.Success(fmt.Sprintf("✅ Found %d matching VMs", len(matches)))
	}

	// Output format
	if count {
		// Just print count
		fmt.Println(len(matches))
	} else if namesOnly {
		// Print VM names only (one per line)
		for _, vm := range matches {
			fmt.Println(vm.Name)
		}
	} else if jsonOutput {
		// JSON output
		output, _ := json.MarshalIndent(map[string]interface{}{
			"vms":       matches,
			"total":     len(matches),
			"pattern":   pattern,
			"field":     field,
			"timestamp": time.Now(),
		}, "", "  ")
		fmt.Println(string(output))
	} else {
		// Display in table format
		if len(matches) == 0 {
			pterm.Info.Println("No matching VMs found")
			return
		}
		displayVMs(matches)
	}
}

func handleRipgrep(daemonURL string, patterns []string, smartCase, ignoreCase, invert, count, countMatches,
	namesOnly, filesWithoutMatch bool, color string, stats, multiline bool, maxCount int, jsonOutput bool) {

	startTime := time.Now()

	// Only show spinner if not in JSON/count/names-only mode
	var spinner *pterm.SpinnerPrinter
	if !jsonOutput && !count && !namesOnly && !filesWithoutMatch && !stats {
		spinner, _ = pterm.DefaultSpinner.Start("🔍 Searching VMs with ripgrep...")
	}

	// Fetch all VMs
	resp, err := apiRequestWithTimeout(daemonURL+"/vms/list", "GET", "", nil, 120*time.Second)
	if err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to fetch VMs: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		}
		os.Exit(1)
	}

	var vmResp struct {
		VMs       []vsphere.VMInfo `json:"vms"`
		Total     int              `json:"total"`
		Timestamp time.Time        `json:"timestamp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
		if spinner != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Compile all patterns
	type matcherInfo struct {
		pattern string
		matcher func(string) bool
	}
	var matchers []matcherInfo

	for _, pattern := range patterns {
		// Smart case: if pattern is all lowercase, ignore case
		shouldIgnoreCase := ignoreCase
		if smartCase && !ignoreCase {
			isAllLower := true
			for _, r := range pattern {
				if r >= 'A' && r <= 'Z' {
					isAllLower = false
					break
				}
			}
			if isAllLower {
				shouldIgnoreCase = true
			}
		}

		// Compile regex pattern
		flags := ""
		if shouldIgnoreCase {
			flags = "(?i)"
		}
		if multiline {
			flags += "(?s)" // . matches newlines
		}

		re, err := regexp.Compile(flags + pattern)
		if err != nil {
			pterm.Error.Printfln("Invalid regex pattern %q: %v", pattern, err)
			os.Exit(1)
		}

		matchers = append(matchers, matcherInfo{
			pattern: pattern,
			matcher: re.MatchString,
		})
	}

	// Statistics tracking
	var statsData struct {
		VMsSearched    int
		VMsMatched     int
		TotalMatches   int
		Patterns       []string
		ElapsedTime    time.Duration
		BytesSearched  int64
		MatchesByField map[string]int
	}
	statsData.Patterns = patterns
	statsData.MatchesByField = make(map[string]int)

	// Search all VMs
	var matches []matchResult
	matchedCount := 0

	for _, vm := range vmResp.VMs {
		statsData.VMsSearched++

		// Build searchable text from all VM fields
		searchText := fmt.Sprintf("%s\n%s\n%s\n%s",
			vm.Name, vm.Path, vm.GuestOS, vm.PowerState)
		statsData.BytesSearched += int64(len(searchText))

		// Check if any pattern matches
		vmMatched := false
		totalMatchesInVM := 0
		matchedFields := make(map[string]bool)

		for _, m := range matchers {
			// Check each field separately to track which fields matched
			fields := map[string]string{
				"name":  vm.Name,
				"path":  vm.Path,
				"os":    vm.GuestOS,
				"power": vm.PowerState,
			}

			for fieldName, fieldValue := range fields {
				if m.matcher(fieldValue) {
					vmMatched = true
					matchedFields[fieldName] = true
					if countMatches {
						// Count all occurrences
						allMatches := regexp.MustCompile(m.pattern).FindAllString(fieldValue, -1)
						totalMatchesInVM += len(allMatches)
					}
				}
			}
		}

		// Apply invert flag
		if invert {
			vmMatched = !vmMatched
		}

		if vmMatched {
			var fieldList []string
			for field := range matchedFields {
				fieldList = append(fieldList, field)
				statsData.MatchesByField[field]++
			}

			matches = append(matches, matchResult{
				vm:            vm,
				matchCount:    totalMatchesInVM,
				matchedFields: fieldList,
			})
			statsData.VMsMatched++
			statsData.TotalMatches += totalMatchesInVM

			matchedCount++
			if maxCount > 0 && matchedCount >= maxCount {
				break
			}
		}
	}

	statsData.ElapsedTime = time.Since(startTime)

	if spinner != nil {
		spinner.Success(fmt.Sprintf("✅ Found %d matching VMs", len(matches)))
	}

	// Output based on flags
	if stats {
		// Show statistics
		pterm.Println()
		pterm.DefaultSection.Println("📊 Search Statistics")
		pterm.Println()

		statLines := [][]string{
			{"Metric", "Value"},
			{"Patterns", strings.Join(statsData.Patterns, ", ")},
			{"VMs Searched", fmt.Sprintf("%d", statsData.VMsSearched)},
			{"VMs Matched", fmt.Sprintf("%d", statsData.VMsMatched)},
			{"Total Matches", fmt.Sprintf("%d", statsData.TotalMatches)},
			{"Bytes Searched", formatBytes(statsData.BytesSearched)},
			{"Elapsed Time", statsData.ElapsedTime.String()},
			{"Search Speed", fmt.Sprintf("%.2f MB/s", float64(statsData.BytesSearched)/statsData.ElapsedTime.Seconds()/1024/1024)},
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed().
			WithData(statLines).
			Render()

		pterm.Println()
		pterm.DefaultSection.Println("📋 Matches by Field")
		pterm.Println()

		fieldLines := [][]string{{"Field", "Matches"}}
		for field, count := range statsData.MatchesByField {
			fieldLines = append(fieldLines, []string{field, fmt.Sprintf("%d", count)})
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed().
			WithData(fieldLines).
			Render()
		return
	}

	if count {
		// Just print VM count
		fmt.Println(len(matches))
		return
	}

	if countMatches {
		// Print total matches count
		fmt.Println(statsData.TotalMatches)
		return
	}

	if namesOnly {
		// Print VM names only (one per line)
		for _, m := range matches {
			fmt.Println(m.vm.Name)
		}
		return
	}

	if filesWithoutMatch {
		// Print VMs that don't match
		for _, vm := range vmResp.VMs {
			found := false
			for _, m := range matches {
				if m.vm.Name == vm.Name {
					found = true
					break
				}
			}
			if !found {
				fmt.Println(vm.Name)
			}
		}
		return
	}

	if jsonOutput {
		// JSON output
		output, _ := json.MarshalIndent(map[string]interface{}{
			"vms":       extractVMs(matches),
			"total":     len(matches),
			"patterns":  patterns,
			"stats":     statsData,
			"timestamp": time.Now(),
		}, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Default: Display in table format
	if len(matches) == 0 {
		pterm.Info.Println("No matching VMs found")
		return
	}

	// Determine if we should colorize
	useColor := (color == "always") || (color == "auto" && isatty())

	if useColor {
		// Show colorized output with match highlights
		displayVMsWithHighlight(extractVMs(matches), patterns[0])
	} else {
		displayVMs(extractVMs(matches))
	}
}

// Helper to extract VMs from match results
func extractVMs(matches []matchResult) []vsphere.VMInfo {
	vms := make([]vsphere.VMInfo, len(matches))
	for i, m := range matches {
		vms[i] = m.vm
	}
	return vms
}

// Display VMs with pattern highlighting
func displayVMsWithHighlight(vms []vsphere.VMInfo, pattern string) {
	// Calculate statistics
	var poweredOn, poweredOff int
	for _, vm := range vms {
		if vm.PowerState == "poweredOn" {
			poweredOn++
		} else {
			poweredOff++
		}
	}

	// Summary
	pterm.Println()
	summary := fmt.Sprintf("Found %d VMs | 🟢 %d ON | 🔴 %d OFF",
		len(vms), poweredOn, poweredOff)
	pterm.Info.Println(summary)
	pterm.Println()

	// Table header
	data := [][]string{
		{"Name", "Path", "Power", "OS", "CPUs", "Memory (GB)", "Storage"},
	}

	// Compile pattern for highlighting
	highlightRe := regexp.MustCompile("(?i)" + regexp.QuoteMeta(pattern))

	for _, vm := range vms {
		powerIcon := "🔴"
		if vm.PowerState == "poweredOn" {
			powerIcon = "🟢"
		}

		// Highlight matches in name
		name := vm.Name
		if highlightRe.MatchString(name) {
			name = pterm.FgLightGreen.Sprint(name)
		}

		data = append(data, []string{
			name,
			truncate(vm.Path, 40),
			powerIcon,
			truncate(vm.GuestOS, 20),
			fmt.Sprintf("%d", vm.NumCPU),
			fmt.Sprintf("%.1f", float64(vm.MemoryMB)/1024),
			formatBytes(vm.Storage),
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func displayVMs(vms []vsphere.VMInfo) {
	// Calculate statistics
	var poweredOn, poweredOff, totalMemory, totalCPUs int
	var totalStorage int64

	for _, vm := range vms {
		if strings.Contains(strings.ToLower(vm.PowerState), "on") {
			poweredOn++
		} else {
			poweredOff++
		}
		totalMemory += int(vm.MemoryMB)
		totalCPUs += int(vm.NumCPU)
		totalStorage += vm.Storage
	}

	// Show summary
	pterm.DefaultSection.Println("📊 VM Summary")
	summaryData := [][]string{
		{"🖥️  Total VMs", fmt.Sprintf("%d", len(vms))},
		{"✅ Powered On", pterm.Green(fmt.Sprintf("%d", poweredOn))},
		{"⭕ Powered Off", pterm.Gray(fmt.Sprintf("%d", poweredOff))},
		{"💾 Total Memory", fmt.Sprintf("%.1f GB", float64(totalMemory)/1024)},
		{"⚡ Total CPUs", fmt.Sprintf("%d", totalCPUs)},
		{"💿 Total Storage", formatBytes(totalStorage)},
	}

	pterm.DefaultTable.
		WithBoxed().
		WithData(summaryData).
		Render()

	pterm.Println()

	// Show VMs table
	pterm.DefaultSection.Println("💻 Virtual Machines")

	data := [][]string{
		{"#", "Name", "Power", "CPU", "Memory", "Storage", "Guest OS"},
	}

	for i, vm := range vms {
		// Truncate name if too long
		name := vm.Name
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		// Color code power state
		power := vm.PowerState
		if strings.Contains(strings.ToLower(power), "on") {
			power = pterm.Green(power)
		} else {
			power = pterm.Gray(power)
		}

		// Truncate guest OS
		guestOS := vm.GuestOS
		if len(guestOS) > 30 {
			guestOS = guestOS[:27] + "..."
		}

		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			name,
			power,
			fmt.Sprintf("%d", vm.NumCPU),
			fmt.Sprintf("%.1f GB", float64(vm.MemoryMB)/1024),
			formatBytes(vm.Storage),
			guestOS,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()
	pterm.Info.Printfln("💡 Tip: Use 'hyperctl list -json' for machine-readable output")
	pterm.Info.Printfln("💡 Tip: Use 'hyperctl list -filter <name>' to filter VMs")

	// Show random fun fact
	showRandomFact()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func handleVM(daemonURL, operation, vmPath string, timeout int) {
	if operation == "" {
		pterm.Error.Println("Operation required (-op)")
		pterm.Info.Println("Available operations: shutdown, poweroff, remove-cdrom, info")
		os.Exit(1)
	}

	if vmPath == "" {
		pterm.Error.Println("VM path required (-path)")
		os.Exit(1)
	}

	var endpoint string
	var opEmoji string
	var opName string

	switch operation {
	case "shutdown":
		endpoint = "/vms/shutdown"
		opEmoji = "🔌"
		opName = "Shutting down"
	case "poweroff":
		endpoint = "/vms/poweroff"
		opEmoji = "⚡"
		opName = "Powering off"
	case "remove-cdrom":
		endpoint = "/vms/remove-cdrom"
		opEmoji = "💿"
		opName = "Removing CD/DVD from"
	case "info":
		endpoint = "/vms/info"
		opEmoji = "ℹ️"
		opName = "Getting info for"
	default:
		pterm.Error.Printfln("Unknown operation: %s", operation)
		pterm.Info.Println("Available operations: shutdown, poweroff, remove-cdrom, info")
		os.Exit(1)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("%s %s VM: %s", opEmoji, opName, vmPath))

	// Prepare request
	reqBody := map[string]interface{}{
		"vm_path": vmPath,
	}
	if operation == "shutdown" && timeout > 0 {
		reqBody["timeout"] = timeout
	}

	data, _ := json.Marshal(reqBody)

	// Make request with appropriate timeout
	reqTimeout := 30 * time.Second
	if operation == "shutdown" {
		reqTimeout = time.Duration(timeout+10) * time.Second
	}

	resp, err := apiRequestWithTimeout(daemonURL+endpoint, "POST", "application/json", data, reqTimeout)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to %s: %v", operation, err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	// Parse response
	if operation == "info" {
		var infoResp struct {
			Success   bool           `json:"success"`
			VMInfo    vsphere.VMInfo `json:"vm_info"`
			Timestamp time.Time      `json:"timestamp"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
			os.Exit(1)
		}

		spinner.Success("Retrieved VM info")

		// Display VM info
		pterm.DefaultSection.Println("📋 VM Information")
		data := [][]string{
			{"Property", "Value"},
			{"Name", infoResp.VMInfo.Name},
			{"Path", infoResp.VMInfo.Path},
			{"Power State", colorizeStatus(infoResp.VMInfo.PowerState)},
			{"Guest OS", infoResp.VMInfo.GuestOS},
			{"CPUs", fmt.Sprintf("%d", infoResp.VMInfo.NumCPU)},
			{"Memory", fmt.Sprintf("%.1f GB", float64(infoResp.VMInfo.MemoryMB)/1024)},
			{"Storage", formatBytes(infoResp.VMInfo.Storage)},
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithHeaderRowSeparator("-").
			WithBoxed().
			WithData(data).
			Render()
	} else {
		var opResp struct {
			Success   bool      `json:"success"`
			Message   string    `json:"message"`
			Timestamp time.Time `json:"timestamp"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&opResp); err != nil {
			spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
			os.Exit(1)
		}

		spinner.Success(opResp.Message)

		// Show success celebration
		showSuccessArt()
		pterm.Success.Printfln("✅ %s", opResp.Message)

		// Show motivation
		showMotivation()

		// Show helpful next steps
		pterm.Println()
		switch operation {
		case "shutdown":
			pterm.Info.Println("💡 Next steps:")
			pterm.Println("   1. Verify VM is powered off: hyperctl vm -op info -path " + vmPath)
			pterm.Println("   2. Remove CD/DVD: hyperctl vm -op remove-cdrom -path " + vmPath)
			pterm.Println("   3. Export VM: hyperctl submit -vm " + vmPath + " -output /tmp/export")
		case "poweroff":
			pterm.Info.Println("💡 Next steps:")
			pterm.Println("   1. Remove CD/DVD: hyperctl vm -op remove-cdrom -path " + vmPath)
			pterm.Println("   2. Export VM: hyperctl submit -vm " + vmPath + " -output /tmp/export")
		case "remove-cdrom":
			pterm.Info.Println("💡 Next steps:")
			pterm.Println("   1. Export VM: hyperctl submit -vm " + vmPath + " -output /tmp/export")
			pterm.Println("   2. Monitor export: hyperctl query -status running")
		}
	}
}

func handleStatus(daemonURL string) {
	spinner, _ := pterm.DefaultSpinner.Start("📊 Getting daemon status...")

	resp, err := apiRequest(daemonURL+"/status", "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to get status: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	var status models.DaemonStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success("Retrieved daemon status")
	pterm.Println()

	// Show connection info
	pterm.Info.Printfln("🔗 Connected to: %s", daemonURL)
	pterm.Println()

	// Display main status
	pterm.DefaultSection.Println("📊 Daemon Status")
	statusData := [][]string{
		{"Property", "Value"},
		{"⚙️  Version", status.Version},
		{"⏱️  Uptime", status.Uptime},
		{"📍 URL", daemonURL},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(statusData).
		Render()

	pterm.Println()

	// Display job statistics
	pterm.DefaultSection.Println("📈 Job Statistics")

	// Calculate percentages
	total := float64(status.TotalJobs)
	runningPct := 0.0
	completedPct := 0.0
	failedPct := 0.0
	if total > 0 {
		runningPct = float64(status.RunningJobs) / total * 100
		completedPct = float64(status.CompletedJobs) / total * 100
		failedPct = float64(status.FailedJobs) / total * 100
	}

	jobsData := [][]string{
		{"Status", "Count", "Percentage"},
		{"📦 Total Jobs", fmt.Sprintf("%d", status.TotalJobs), "100%"},
		{"🔄 Running", pterm.LightCyan(fmt.Sprintf("%d", status.RunningJobs)), fmt.Sprintf("%.1f%%", runningPct)},
		{"✅ Completed", pterm.Green(fmt.Sprintf("%d", status.CompletedJobs)), fmt.Sprintf("%.1f%%", completedPct)},
		{"❌ Failed", pterm.Red(fmt.Sprintf("%d", status.FailedJobs)), fmt.Sprintf("%.1f%%", failedPct)},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(jobsData).
		Render()

	pterm.Println()

	// Show helpful actions
	if status.RunningJobs > 0 {
		pterm.Info.Println("💡 You have running jobs:")
		pterm.Println("   View them: hyperctl query -status running")
	} else if status.TotalJobs == 0 {
		pterm.Info.Println("💡 No jobs yet. Start by:")
		pterm.Println("   1. List VMs: hyperctl list")
		pterm.Println("   2. Submit job: hyperctl submit -vm /data/vm/my-vm -output /tmp/export")
	} else {
		pterm.Success.Println("✅ All jobs completed!")
	}
}

func handleCancel(daemonURL, jobIDs string) {
	if jobIDs == "" {
		pterm.Error.Println("Job ID required (-id)")
		os.Exit(1)
	}

	spinner, _ := pterm.DefaultSpinner.Start("Cancelling job(s)...")

	req := models.CancelRequest{
		JobIDs: strings.Split(jobIDs, ","),
	}

	data, _ := json.Marshal(req)
	resp, err := apiRequest(daemonURL+"/jobs/cancel", "POST", "application/json", data)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to cancel: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	var cancelResp models.CancelResponse
	if err := json.NewDecoder(resp.Body).Decode(&cancelResp); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success(fmt.Sprintf("Cancelled %d job(s)", len(cancelResp.Cancelled)))

	if len(cancelResp.Cancelled) > 0 {
		pterm.Success.Println("Cancelled:")
		for _, id := range cancelResp.Cancelled {
			pterm.Info.Printfln("  - %s", id)
		}
	}

	if len(cancelResp.Failed) > 0 {
		pterm.Warning.Println("Failed to cancel:")
		for _, id := range cancelResp.Failed {
			pterm.Error.Printfln("  - %s: %s", id, cancelResp.Errors[id])
		}
	}
}

func displayJobs(jobs []*models.Job) {
	data := [][]string{
		{"Job ID", "Name", "VM Path", "Status", "Progress", "Started"},
	}

	for _, job := range jobs {
		id := job.Definition.ID
		if len(id) > 8 {
			id = id[:8] + "..."
		}

		vmPath := job.Definition.VMPath
		if len(vmPath) > 30 {
			vmPath = "..." + vmPath[len(vmPath)-27:]
		}

		status := colorizeStatus(string(job.Status))

		progress := "-"
		if job.Progress != nil {
			progress = fmt.Sprintf("%s (%.1f%%)",
				job.Progress.Phase,
				job.Progress.PercentComplete)
		}

		started := "-"
		if job.StartedAt != nil {
			started = job.StartedAt.Format("15:04:05")
		}

		data = append(data, []string{
			id,
			job.Definition.Name,
			vmPath,
			status,
			progress,
			started,
		})
	}

	pterm.DefaultSection.Println("Jobs")
	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

func apiRequest(url, method, contentType string, body []byte) (*http.Response, error) {
	return apiRequestWithTimeout(url, method, contentType, body, 30*time.Second)
}

func apiRequestWithTimeout(url, method, contentType string, body []byte, timeout time.Duration) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{Timeout: timeout}
	return client.Do(req)
}

func handleCompletion(shellType string) {
	if shellType == "" {
		pterm.Error.Println("Shell type required. Use: -shell bash|zsh|fish")
		pterm.Println()
		pterm.Info.Println("Examples:")
		pterm.Println("  hyperctl completion -shell bash")
		pterm.Println("  hyperctl completion -shell zsh")
		pterm.Println("  hyperctl completion -shell fish")
		pterm.Println()

		// Show installation instructions for all shells
		for _, shell := range completion.SupportedShells() {
			pterm.DefaultSection.Printfln("%s Installation", strings.ToUpper(string(shell)))
			pterm.Println(completion.InstallInstructions(shell))
		}
		os.Exit(1)
	}

	shell := completion.Shell(shellType)
	script, err := completion.Generate(shell)
	if err != nil {
		pterm.Error.Printfln("Error generating completion: %v", err)
		pterm.Info.Println("Supported shells: bash, zsh, fish")
		os.Exit(1)
	}

	// Output the completion script
	fmt.Println(script)
}

// GenerateExampleJobFile creates an example job file
func GenerateExampleJobFile(format string) {
	job := models.JobDefinition{
		Name:       "example-export",
		VMPath:     "/datacenter/vm/my-vm",
		OutputPath: "/tmp/export-my-vm",
		Options: &models.ExportOptions{
			ParallelDownloads:      4,
			RemoveCDROM:            true,
			ShowIndividualProgress: false,
		},
	}

	var data []byte
	var err error
	var filename string

	if format == "yaml" {
		data, err = yaml.Marshal(job)
		filename = "example-job.yaml"
	} else {
		data, err = json.MarshalIndent(job, "", "  ")
		filename = "example-job.json"
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", filename)
}
