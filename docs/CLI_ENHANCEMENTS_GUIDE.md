# CLI Enhancements Guide

Guide for enhancing HyperSDK CLI with advanced features.

**Version**: v2.2.0
**Status**: Enhancement Roadmap

---

## Current CLI Capabilities

### ✅ Implemented Features

1. **Manifest Generation**
   - All VM commands generate valid YAML manifests
   - Can pipe directly to kubectl

2. **Basic Operations**
   - Create, start, stop, restart VMs
   - Clone, migrate, resize operations
   - Snapshot management
   - Template operations

3. **Output Formats**
   - YAML output (default)
   - JSON output (via -output json)

### Example Usage

```bash
# Generate VM manifest
hyperctl k8s -op vm-create -vm my-vm -cpus 4 -memory 8Gi

# Apply to cluster
hyperctl k8s -op vm-create -vm my-vm -cpus 4 -memory 8Gi | kubectl apply -f -

# Create with JSON output
hyperctl k8s -op vm-create -vm my-vm -cpus 4 -memory 8Gi -output json
```

---

## 🚀 Recommended Enhancements

### 1. Advanced Filtering

**Implementation Priority**: Medium
**Estimated Effort**: 2-3 hours

#### Proposed Features

```bash
# Filter by status
hyperctl k8s vm list --status running
hyperctl k8s vm list --status stopped,failed

# Filter by node
hyperctl k8s vm list --node node-1

# Filter by resource requirements
hyperctl k8s vm list --min-cpus 4
hyperctl k8s vm list --min-memory 8Gi

# Combined filters
hyperctl k8s vm list --status running --node node-1 --namespace production

# Label selectors
hyperctl k8s vm list --selector app=web
hyperctl k8s vm list -l environment=production,tier=frontend
```

#### Implementation Approach

1. Add filter flags to `main.go`:
   ```go
   k8sVMStatus := k8sCmd.String("status", "", "Filter by status (running,stopped,failed)")
   k8sVMNode := k8sCmd.String("node", "", "Filter by node name")
   k8sVMLabels := k8sCmd.String("selector", "", "Label selector")
   ```

2. Implement in `vm_commands.go`:
   ```go
   func handleVMList(filters VMFilters) {
       client := NewK8sClient(kubeconfig, namespace)
       vms, err := client.ListVMs(filters)
       // Format and display
   }
   ```

3. Use Kubernetes label selectors:
   ```go
   listOptions := metav1.ListOptions{
       LabelSelector: filters.Labels,
   }
   ```

### 2. Watch Mode

**Implementation Priority**: High
**Estimated Effort**: 1-2 hours

#### Proposed Features

```bash
# Watch VMs in real-time
hyperctl k8s vm list --watch

# Watch specific VM
hyperctl k8s vm get my-vm --watch

# Watch with updates only
hyperctl k8s vm list --watch --output json
```

#### Implementation Approach

```go
func watchVMs(client *K8sClient, filters VMFilters) error {
    watcher, err := client.Watch(filters)
    if err != nil {
        return err
    }
    defer watcher.Stop()

    for event := range watcher.ResultChan() {
        vm := event.Object.(*VirtualMachine)
        printVMUpdate(event.Type, vm)
    }
    return nil
}
```

### 3. Progress Bars

**Implementation Priority**: Medium
**Estimated Effort**: 2 hours

#### Proposed Features

```bash
# Show progress for operations
hyperctl k8s vm clone my-vm --target clone --wait --show-progress

# Output with progress bar
Creating VM clone...
[████████████████░░░░] 80% - Copying disk 2 of 3
```

#### Implementation Approach

Use existing `pterm` library:

```go
func waitForOperation(vmOpName string) error {
    p := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Operation Progress")
    p.Start()

    for {
        vmOp, err := getVMOperation(vmOpName)
        if err != nil {
            return err
        }

        p.UpdateTitle(vmOp.Status.Message)
        p.Update(int(vmOp.Status.Progress))

        if vmOp.Status.Phase == "Succeeded" || vmOp.Status.Phase == "Failed" {
            break
        }
        time.Sleep(2 * time.Second)
    }

    p.Stop()
    return nil
}
```

### 4. JSON/YAML Output Everywhere

**Implementation Priority**: Low
**Estimated Effort**: 1 hour

#### Proposed Features

```bash
# List with JSON
hyperctl k8s vm list -o json

# Get with YAML
hyperctl k8s vm get my-vm -o yaml

# Export to file
hyperctl k8s vm get my-vm -o yaml > my-vm.yaml
```

#### Implementation

```go
func outputVMList(vms []VM, format string) {
    switch format {
    case "json":
        data, _ := json.MarshalIndent(vms, "", "  ")
        fmt.Println(string(data))
    case "yaml":
        data, _ := yaml.Marshal(vms)
        fmt.Println(string(data))
    case "table", "":
        printVMTable(vms)
    }
}
```

### 5. Interactive Mode

**Implementation Priority**: Low
**Estimated Effort**: 3-4 hours

#### Proposed Features

```bash
# Interactive VM creation
hyperctl k8s vm create --interactive

# Wizard-style prompts:
# VM Name: my-vm
# CPUs [2]: 4
# Memory [4Gi]: 8Gi
# Template [none]: ubuntu-22-04
# Create? [Y/n]: y
```

#### Implementation

Use `pterm` interactive select:

```go
func interactiveVMCreate() error {
    name, _ := pterm.DefaultInteractiveTextInput.Show("VM Name")

    cpus, _ := pterm.DefaultInteractiveSelect.
        WithOptions([]string{"2", "4", "8", "16"}).
        Show("CPUs")

    memory, _ := pterm.DefaultInteractiveSelect.
        WithOptions([]string{"4Gi", "8Gi", "16Gi", "32Gi"}).
        Show("Memory")

    templates, _ := listTemplates()
    template, _ := pterm.DefaultInteractiveSelect.
        WithOptions(templates).
        Show("Template")

    // Create VM with collected parameters
}
```

---

## 🛠️ Quick Workarounds (Current)

Until advanced features are implemented, use these patterns:

### Filtering

```bash
# Filter with kubectl
kubectl get vm -l app=web
kubectl get vm --field-selector status.phase=Running

# Filter with jq
hyperctl k8s vm list -o json | jq '.[] | select(.status.phase == "Running")'
```

### Watch Mode

```bash
# Use kubectl watch
kubectl get vm --watch

# Watch specific VM
kubectl get vm my-vm --watch -o yaml
```

### Progress Tracking

```bash
# Poll for status
while true; do
  kubectl get vm my-vm -o jsonpath='{.status.phase}'
  sleep 2
done

# Watch VMOperation
kubectl get vmoperation clone-my-vm --watch
```

---

## 📋 Implementation Checklist

When implementing enhancements:

- [ ] Add flags to `main.go`
- [ ] Update usage/help text
- [ ] Implement in `vm_commands.go`
- [ ] Add unit tests
- [ ] Update documentation
- [ ] Add examples to README
- [ ] Test with live cluster

---

## 🔍 Testing Enhanced CLI

```bash
# Test filtering
hyperctl k8s vm list --status running
hyperctl k8s vm list --node node-1

# Test watch mode
hyperctl k8s vm list --watch &
kubectl apply -f vm.yaml
# Should see live updates

# Test progress
hyperctl k8s vm clone my-vm --target clone --wait --show-progress

# Test output formats
hyperctl k8s vm list -o json
hyperctl k8s vm list -o yaml
hyperctl k8s vm list -o table
```

---

## 💡 Best Practices

1. **Consistent Flag Names**
   - Use kubectl-style flags where possible
   - `-o` for output, `--watch` for watch mode
   - `--selector` or `-l` for label selectors

2. **Error Handling**
   - Clear error messages
   - Suggest fixes (e.g., "try kubectl get nodes")
   - Graceful degradation

3. **Performance**
   - Cache cluster information
   - Paginate large result sets
   - Stream watch events efficiently

4. **User Experience**
   - Color-coded output (use pterm)
   - Progress indicators for slow operations
   - Smart defaults (current namespace, etc.)

---

## 📚 Related Documentation

- [VM Management Guide](VM_MANAGEMENT.md)
- [hyperctl Command Reference](../README.md)
- [kubectl Documentation](https://kubernetes.io/docs/reference/kubectl/)

---

**HyperSDK CLI Enhancements Guide**
Version: v2.2.0
Status: Enhancement Roadmap
