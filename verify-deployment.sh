#!/bin/bash
#
# HyperSDK Deployment Verification Script
# Validates that HyperSDK Kubernetes operator and VMs are correctly installed
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Print functions
print_header() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_test() {
    echo -e "\n${YELLOW}▶${NC} $1"
}

print_success() {
    echo -e "  ${GREEN}✓${NC} $1"
    ((PASSED++))
}

print_error() {
    echo -e "  ${RED}✗${NC} $1"
    ((FAILED++))
}

print_warning() {
    echo -e "  ${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

print_info() {
    echo -e "  ${BLUE}ℹ${NC} $1"
}

# Test functions
test_kubectl() {
    print_test "Checking kubectl installation"
    if command -v kubectl &> /dev/null; then
        local version=$(kubectl version --client --short 2>/dev/null | grep -oP 'v\K[0-9.]+' | head -1)
        print_success "kubectl is installed (version: ${version:-unknown})"
        return 0
    else
        print_error "kubectl is not installed"
        return 1
    fi
}

test_cluster_access() {
    print_test "Checking Kubernetes cluster access"
    if kubectl cluster-info &> /dev/null; then
        local context=$(kubectl config current-context 2>/dev/null)
        print_success "Connected to cluster (context: $context)"
        return 0
    else
        print_error "Cannot connect to Kubernetes cluster"
        return 1
    fi
}

test_hypersdk_namespace() {
    print_test "Checking HyperSDK namespace"
    if kubectl get namespace hypersdk-system &> /dev/null; then
        print_success "Namespace 'hypersdk-system' exists"
        return 0
    else
        print_warning "Namespace 'hypersdk-system' does not exist"
        print_info "Run: kubectl create namespace hypersdk-system"
        return 1
    fi
}

test_crds() {
    print_test "Checking Custom Resource Definitions (CRDs)"
    local required_crds=(
        "virtualmachines.hypersdk.io"
        "vmoperations.hypersdk.io"
        "vmsnapshots.hypersdk.io"
        "vmtemplates.hypersdk.io"
    )

    local missing_crds=()
    for crd in "${required_crds[@]}"; do
        if kubectl get crd "$crd" &> /dev/null; then
            print_success "CRD '$crd' is installed"
        else
            print_error "CRD '$crd' is missing"
            missing_crds+=("$crd")
        fi
    done

    if [ ${#missing_crds[@]} -eq 0 ]; then
        return 0
    else
        print_info "Install missing CRDs: kubectl apply -f deploy/crds/"
        return 1
    fi
}

test_operator() {
    print_test "Checking HyperSDK Operator deployment"
    if kubectl get deployment -n hypersdk-system hypersdk-operator &> /dev/null; then
        local ready=$(kubectl get deployment -n hypersdk-system hypersdk-operator -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
        local desired=$(kubectl get deployment -n hypersdk-system hypersdk-operator -o jsonpath='{.spec.replicas}' 2>/dev/null)

        if [ "$ready" == "$desired" ] && [ "$ready" != "" ]; then
            print_success "Operator is running ($ready/$desired replicas ready)"
            return 0
        else
            print_warning "Operator exists but not all replicas ready ($ready/$desired)"
            return 1
        fi
    else
        print_error "Operator deployment not found"
        print_info "Install operator: helm install hypersdk-operator ./deploy/helm/hypersdk-operator -n hypersdk-system"
        return 1
    fi
}

test_operator_logs() {
    print_test "Checking operator logs for errors"
    if kubectl get pods -n hypersdk-system -l app=hypersdk-operator &> /dev/null; then
        local pod=$(kubectl get pods -n hypersdk-system -l app=hypersdk-operator -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

        if [ -n "$pod" ]; then
            local errors=$(kubectl logs -n hypersdk-system "$pod" --tail=100 2>/dev/null | grep -i "error\|fatal\|panic" | wc -l)

            if [ "$errors" -eq 0 ]; then
                print_success "No errors in operator logs"
                return 0
            else
                print_warning "Found $errors error lines in operator logs"
                print_info "Check logs: kubectl logs -n hypersdk-system -l app=hypersdk-operator"
                return 1
            fi
        else
            print_warning "Cannot find operator pod"
            return 1
        fi
    else
        print_warning "Operator pods not found"
        return 1
    fi
}

test_rbac() {
    print_test "Checking RBAC configuration"
    local has_sa=false
    local has_role=false
    local has_binding=false

    if kubectl get serviceaccount -n hypersdk-system hypersdk-operator &> /dev/null; then
        print_success "ServiceAccount exists"
        has_sa=true
    else
        print_warning "ServiceAccount not found"
    fi

    if kubectl get clusterrole hypersdk-operator &> /dev/null; then
        print_success "ClusterRole exists"
        has_role=true
    else
        print_warning "ClusterRole not found"
    fi

    if kubectl get clusterrolebinding hypersdk-operator &> /dev/null; then
        print_success "ClusterRoleBinding exists"
        has_binding=true
    else
        print_warning "ClusterRoleBinding not found"
    fi

    if $has_sa && $has_role && $has_binding; then
        return 0
    else
        print_info "RBAC is created by Helm chart during installation"
        return 1
    fi
}

test_dashboard_service() {
    print_test "Checking dashboard service"
    if kubectl get service -n hypersdk-system hypersdk-operator &> /dev/null; then
        local port=$(kubectl get service -n hypersdk-system hypersdk-operator -o jsonpath='{.spec.ports[0].port}' 2>/dev/null)
        print_success "Dashboard service exists (port: $port)"
        print_info "Access dashboard: kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080"
        return 0
    else
        print_warning "Dashboard service not found"
        return 1
    fi
}

test_storage_class() {
    print_test "Checking storage class availability"
    local sc_count=$(kubectl get storageclass --no-headers 2>/dev/null | wc -l)

    if [ "$sc_count" -gt 0 ]; then
        local default_sc=$(kubectl get storageclass -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}' 2>/dev/null)

        if [ -n "$default_sc" ]; then
            print_success "Default storage class: $default_sc"
        else
            print_warning "$sc_count storage class(es) found, but no default set"
        fi
        return 0
    else
        print_error "No storage classes found"
        print_info "VMs require a storage class for disk volumes"
        return 1
    fi
}

test_nodes() {
    print_test "Checking cluster nodes"
    local node_count=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
    local ready_nodes=$(kubectl get nodes --no-headers 2>/dev/null | grep -c " Ready" || true)

    if [ "$node_count" -gt 0 ]; then
        print_success "$ready_nodes/$node_count nodes are ready"

        if [ "$ready_nodes" -lt 2 ]; then
            print_warning "VM migration requires at least 2 worker nodes"
        fi
        return 0
    else
        print_error "No nodes found in cluster"
        return 1
    fi
}

test_vms() {
    print_test "Checking virtual machines"
    if kubectl get vm &> /dev/null 2>&1; then
        local vm_count=$(kubectl get vm --no-headers 2>/dev/null | wc -l)

        if [ "$vm_count" -gt 0 ]; then
            print_success "$vm_count VM(s) found"

            local running=$(kubectl get vm --no-headers 2>/dev/null | grep -c "Running" || true)
            local stopped=$(kubectl get vm --no-headers 2>/dev/null | grep -c "Stopped" || true)

            print_info "Running: $running, Stopped: $stopped"
        else
            print_info "No VMs found (this is normal for new installations)"
        fi
        return 0
    else
        print_warning "Cannot query VMs (CRD may not be installed)"
        return 1
    fi
}

test_vm_templates() {
    print_test "Checking VM templates"
    if kubectl get vmtemplate &> /dev/null 2>&1; then
        local template_count=$(kubectl get vmtemplate --no-headers 2>/dev/null | wc -l)

        if [ "$template_count" -gt 0 ]; then
            print_success "$template_count template(s) available"
        else
            print_info "No templates found"
            print_info "Create templates: kubectl apply -f deploy/examples/vmtemplate-ubuntu.yaml"
        fi
        return 0
    else
        print_warning "Cannot query templates (CRD may not be installed)"
        return 1
    fi
}

test_helm() {
    print_test "Checking Helm installation (optional)"
    if command -v helm &> /dev/null; then
        local version=$(helm version --short 2>/dev/null | grep -oP 'v\K[0-9.]+' | head -1)
        print_success "Helm is installed (version: ${version:-unknown})"
        return 0
    else
        print_warning "Helm is not installed (required for operator deployment)"
        print_info "Install Helm: https://helm.sh/docs/intro/install/"
        return 1
    fi
}

print_summary() {
    print_header "Verification Summary"

    local total=$((PASSED + FAILED + WARNINGS))
    echo -e "\n${GREEN}✓ Passed:${NC}   $PASSED"
    echo -e "${RED}✗ Failed:${NC}   $FAILED"
    echo -e "${YELLOW}⚠ Warnings:${NC} $WARNINGS"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if [ "$FAILED" -eq 0 ]; then
        if [ "$WARNINGS" -eq 0 ]; then
            echo -e "\n${GREEN}✓✓✓ All checks passed! HyperSDK is ready to use.${NC}\n"
        else
            echo -e "\n${YELLOW}⚠ Deployment is functional but has some warnings.${NC}"
            echo -e "${YELLOW}⚠ Review the warnings above to optimize your setup.${NC}\n"
        fi
        return 0
    else
        echo -e "\n${RED}✗✗✗ Deployment has critical issues.${NC}"
        echo -e "${RED}✗ Please address the failed checks above.${NC}\n"
        return 1
    fi
}

print_next_steps() {
    print_header "Next Steps"

    if [ "$FAILED" -eq 0 ]; then
        echo -e "\n${GREEN}Your HyperSDK installation is ready!${NC}\n"
        echo "Try these commands:"
        echo ""
        echo "  # Access dashboard"
        echo "  kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080"
        echo "  # Open: http://localhost:8080/k8s/vms"
        echo ""
        echo "  # Create a VM template"
        echo "  kubectl apply -f deploy/examples/vmtemplate-ubuntu.yaml"
        echo ""
        echo "  # Create a VM"
        echo "  kubectl apply -f deploy/examples/vm-ubuntu.yaml"
        echo ""
        echo "  # Check VM status"
        echo "  kubectl get vm"
        echo ""
        echo "For more information, see:"
        echo "  - Quick Start: ./QUICKSTART.md"
        echo "  - VM Management: ./docs/VM_MANAGEMENT.md"
        echo "  - Testing: ./deploy/VM_INTEGRATION_TESTING.md"
    else
        echo -e "\n${RED}Please fix the issues above before proceeding.${NC}\n"
        echo "Common fixes:"
        echo ""
        echo "  # Install CRDs"
        echo "  kubectl apply -f deploy/crds/"
        echo ""
        echo "  # Create namespace"
        echo "  kubectl create namespace hypersdk-system"
        echo ""
        echo "  # Install operator"
        echo "  helm install hypersdk-operator ./deploy/helm/hypersdk-operator \\"
        echo "    --namespace hypersdk-system --create-namespace"
        echo ""
        echo "For complete installation guide, see ./QUICKSTART.md"
    fi
    echo ""
}

# Main execution
main() {
    clear
    print_header "HyperSDK Deployment Verification"
    echo -e "\nThis script checks if HyperSDK is correctly installed on your Kubernetes cluster.\n"

    # Prerequisites
    print_header "Prerequisites"
    test_kubectl || exit 1
    test_helm
    test_cluster_access || exit 1

    # Cluster resources
    print_header "Cluster Resources"
    test_nodes
    test_storage_class

    # HyperSDK installation
    print_header "HyperSDK Installation"
    test_hypersdk_namespace
    test_crds
    test_operator
    test_rbac
    test_dashboard_service

    # Operational checks
    print_header "Operational Checks"
    test_operator_logs
    test_vms
    test_vm_templates

    # Summary and next steps
    echo ""
    print_summary
    print_next_steps

    # Exit code based on failures
    if [ "$FAILED" -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main "$@"
