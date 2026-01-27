#!/bin/bash
# hyper2kvm-demo.sh - Interactive demo of HyperSDK → hyper2kvm workflow
# Usage: ./hyper2kvm-demo.sh [--daemon-url http://localhost:8080]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DAEMON_URL="${1:-http://localhost:8080}"
PROVIDER="vsphere"
EXPORT_BASE="/tmp/hypersdk-demo"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo "=============================================="
    echo "$1"
    echo "=============================================="
    echo ""
}

check_dependencies() {
    log_info "Checking dependencies..."

    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq is not installed (needed for JSON parsing)"
        exit 1
    fi

    log_success "All dependencies available"
}

check_daemon() {
    log_info "Checking HyperSDK daemon at ${DAEMON_URL}..."

    if ! HEALTH=$(curl -sf ${DAEMON_URL}/health 2>&1); then
        log_error "Cannot connect to HyperSDK daemon at ${DAEMON_URL}"
        log_warning "Make sure hypervisord is running: systemctl status hypervisord"
        exit 1
    fi

    VERSION=$(echo ${HEALTH} | jq -r '.version // "unknown"')
    log_success "Daemon is healthy (version: ${VERSION})"
}

show_capabilities() {
    print_header "Step 1: Check Daemon Capabilities"

    log_info "Fetching daemon capabilities..."
    CAPS=$(curl -s ${DAEMON_URL}/capabilities)

    echo "${CAPS}" | jq '.' || echo "${CAPS}"

    echo ""
    read -p "Press Enter to continue..."
}

list_vms() {
    print_header "Step 2: List Available VMs"

    log_info "Listing VMs from provider: ${PROVIDER}..."

    VMS=$(curl -s -X POST ${DAEMON_URL}/vms/list \
        -H "Content-Type: application/json" \
        -d "{\"provider\": \"${PROVIDER}\"}" 2>&1)

    if echo "${VMS}" | jq -e '.error' &> /dev/null; then
        log_error "Failed to list VMs: $(echo ${VMS} | jq -r '.error')"
        log_warning "This is expected if provider is not configured"
        VM_LIST="[]"
    else
        VM_LIST="${VMS}"
        echo "${VMS}" | jq '.[] | {name: .name, id: .id, status: .power_state}' || echo "${VMS}"
    fi

    echo ""
    read -p "Press Enter to continue..."
}

select_vm() {
    print_header "Step 3: Select VM to Export"

    if [ "${VM_LIST}" = "[]" ] || [ -z "${VM_LIST}" ]; then
        log_warning "No VMs available from provider"
        log_info "Using demo VM identifier: 'demo-vm-001'"
        VM_ID="demo-vm-001"
        VM_NAME="Demo VM"
    else
        echo "Available VMs:"
        echo "${VM_LIST}" | jq -r '.[] | "\(.id) - \(.name)"'
        echo ""
        read -p "Enter VM ID to export (or press Enter for demo): " INPUT_VM_ID

        if [ -z "${INPUT_VM_ID}" ]; then
            VM_ID="demo-vm-001"
            VM_NAME="Demo VM"
        else
            VM_ID="${INPUT_VM_ID}"
            VM_NAME=$(echo "${VM_LIST}" | jq -r ".[] | select(.id == \"${VM_ID}\") | .name")
        fi
    fi

    log_info "Selected VM: ${VM_NAME} (${VM_ID})"
    echo ""
}

configure_export_options() {
    print_header "Step 4: Configure Export Options"

    echo "Select export method:"
    echo "  1) OVA (default)"
    echo "  2) OVF"
    echo "  3) VMDK"
    read -p "Choice [1]: " EXPORT_METHOD_CHOICE

    case ${EXPORT_METHOD_CHOICE:-1} in
        1) EXPORT_METHOD="ova" ;;
        2) EXPORT_METHOD="ovf" ;;
        3) EXPORT_METHOD="vmdk" ;;
        *) EXPORT_METHOD="ova" ;;
    esac

    echo ""
    echo "hyper2kvm integration:"
    echo "  1) Daemon mode (automatic queue processing)"
    echo "  2) Direct mode (immediate conversion)"
    echo "  3) Disabled (export only)"
    read -p "Choice [1]: " INTEGRATION_CHOICE

    case ${INTEGRATION_CHOICE:-1} in
        1)
            HYPER2KVM_ENABLED="true"
            DAEMON_MODE="true"
            AUTO_CONVERT="false"
            ;;
        2)
            HYPER2KVM_ENABLED="true"
            DAEMON_MODE="false"
            AUTO_CONVERT="true"
            ;;
        3)
            HYPER2KVM_ENABLED="false"
            DAEMON_MODE="false"
            AUTO_CONVERT="false"
            ;;
    esac

    log_info "Export method: ${EXPORT_METHOD}"
    log_info "hyper2kvm enabled: ${HYPER2KVM_ENABLED}"
    if [ "${HYPER2KVM_ENABLED}" = "true" ]; then
        log_info "Integration mode: $([ "${DAEMON_MODE}" = "true" ] && echo "Daemon" || echo "Direct")"
    fi

    echo ""
    read -p "Press Enter to submit job..."
}

submit_job() {
    print_header "Step 5: Submit Export Job"

    EXPORT_PATH="${EXPORT_BASE}/${VM_ID}"
    mkdir -p "${EXPORT_PATH}"

    log_info "Submitting export job for VM: ${VM_ID}"
    log_info "Export path: ${EXPORT_PATH}"

    JOB_PAYLOAD=$(cat <<EOF
{
    "vm_identifier": "${VM_ID}",
    "provider": "${PROVIDER}",
    "export_method": "${EXPORT_METHOD}",
    "export_path": "${EXPORT_PATH}",
    "options": {
        "shutdown_vm": false,
        "remove_cdrom": true,
        "compress": true
    },
    "hyper2kvm_integration": {
        "enabled": ${HYPER2KVM_ENABLED},
        "auto_convert": ${AUTO_CONVERT},
        "daemon_mode": ${DAEMON_MODE},
        "instance": "vsphere",
        "output_dir": "/var/lib/kvm/images"
    }
}
EOF
)

    echo "Job payload:"
    echo "${JOB_PAYLOAD}" | jq '.'
    echo ""

    JOB_RESPONSE=$(curl -s -X POST ${DAEMON_URL}/jobs/submit \
        -H "Content-Type: application/json" \
        -d "${JOB_PAYLOAD}")

    if echo "${JOB_RESPONSE}" | jq -e '.error' &> /dev/null; then
        log_error "Job submission failed: $(echo ${JOB_RESPONSE} | jq -r '.error')"
        exit 1
    fi

    JOB_ID=$(echo ${JOB_RESPONSE} | jq -r '.job_id')
    log_success "Job submitted successfully!"
    log_info "Job ID: ${JOB_ID}"

    echo ""
    read -p "Press Enter to monitor progress..."
}

monitor_job() {
    print_header "Step 6: Monitor Job Progress"

    log_info "Monitoring job: ${JOB_ID}"
    log_info "Press Ctrl+C to stop monitoring (job will continue in background)"
    echo ""

    LAST_STATUS=""
    ITERATION=0

    while true; do
        JOB_STATUS=$(curl -s ${DAEMON_URL}/jobs/query?job_id=${JOB_ID})

        if echo "${JOB_STATUS}" | jq -e '.error' &> /dev/null; then
            log_error "Failed to query job: $(echo ${JOB_STATUS} | jq -r '.error')"
            break
        fi

        STATUS=$(echo ${JOB_STATUS} | jq -r '.status // "unknown"')

        # Get progress if available
        PROGRESS_DATA=$(curl -s ${DAEMON_URL}/jobs/progress/${JOB_ID} 2>/dev/null || echo '{}')
        PERCENTAGE=$(echo ${PROGRESS_DATA} | jq -r '.percentage // 0')
        BYTES_TRANSFERRED=$(echo ${PROGRESS_DATA} | jq -r '.bytes_transferred // 0')
        TOTAL_BYTES=$(echo ${PROGRESS_DATA} | jq -r '.total_bytes // 0')

        # Print status update
        if [ "${STATUS}" != "${LAST_STATUS}" ]; then
            echo ""
            log_info "Status changed: ${STATUS}"
            LAST_STATUS="${STATUS}"
        fi

        # Progress bar
        BAR_LENGTH=50
        FILLED_LENGTH=$(( ${PERCENTAGE} * ${BAR_LENGTH} / 100 ))
        BAR=$(printf "%-${BAR_LENGTH}s" "$(printf '#%.0s' $(seq 1 ${FILLED_LENGTH}))")

        printf "\r[${BAR// /-}] ${PERCENTAGE}%% "

        if [ ${TOTAL_BYTES} -gt 0 ]; then
            BYTES_MB=$(( ${BYTES_TRANSFERRED} / 1048576 ))
            TOTAL_MB=$(( ${TOTAL_BYTES} / 1048576 ))
            printf "(${BYTES_MB}/${TOTAL_MB} MB)"
        fi

        # Check if job is complete
        if [ "$STATUS" = "completed" ]; then
            echo ""
            log_success "Export completed successfully!"
            break
        elif [ "$STATUS" = "failed" ]; then
            echo ""
            log_error "Export failed!"
            echo ""
            log_info "Job logs:"
            curl -s ${DAEMON_URL}/jobs/logs/${JOB_ID} | head -n 20
            break
        elif [ "$STATUS" = "cancelled" ]; then
            echo ""
            log_warning "Export was cancelled"
            break
        fi

        # Show detailed status every 10 iterations
        if [ $(( ${ITERATION} % 10 )) -eq 0 ] && [ ${ITERATION} -gt 0 ]; then
            echo ""
            # Get ETA if available
            ETA_DATA=$(curl -s ${DAEMON_URL}/jobs/eta/${JOB_ID} 2>/dev/null || echo '{}')
            ETA=$(echo ${ETA_DATA} | jq -r '.eta // "unknown"')
            if [ "${ETA}" != "unknown" ]; then
                log_info "Estimated completion: ${ETA}"
            fi
        fi

        ITERATION=$(( ${ITERATION} + 1 ))
        sleep 2
    done

    echo ""

    if [ "$STATUS" = "completed" ]; then
        # Show job result
        JOB_RESULT=$(echo ${JOB_STATUS} | jq -r '.result // {}')
        echo ""
        log_info "Export result:"
        echo "${JOB_RESULT}" | jq '.'

        read -p "Press Enter to check hyper2kvm conversion..."
    fi
}

check_hyper2kvm() {
    print_header "Step 7: Check hyper2kvm Integration"

    if [ "${HYPER2KVM_ENABLED}" != "true" ]; then
        log_warning "hyper2kvm integration was not enabled for this job"
        return
    fi

    if [ "${DAEMON_MODE}" = "true" ]; then
        log_info "Checking hyper2kvm daemon status..."

        if systemctl is-active --quiet hyper2kvm@vsphere.service 2>/dev/null; then
            log_success "hyper2kvm daemon is running"
            echo ""
            log_info "Recent daemon logs:"
            journalctl -u hyper2kvm@vsphere.service -n 20 --no-pager || log_warning "Cannot access daemon logs"
        else
            log_warning "hyper2kvm daemon is not running"
            log_info "Start it with: sudo systemctl start hyper2kvm@vsphere.service"
        fi
    else
        log_info "Checking conversion status..."

        CONVERSION_STATUS=$(curl -s ${DAEMON_URL}/conversion-status?job_id=${JOB_ID} 2>/dev/null || echo '{}')

        if echo "${CONVERSION_STATUS}" | jq -e '.status' &> /dev/null; then
            echo "${CONVERSION_STATUS}" | jq '.'
        else
            log_warning "Conversion status not available"
        fi
    fi

    echo ""
    read -p "Press Enter to show final summary..."
}

show_summary() {
    print_header "Summary"

    echo "Job Information:"
    echo "  Job ID: ${JOB_ID}"
    echo "  VM: ${VM_NAME} (${VM_ID})"
    echo "  Export Method: ${EXPORT_METHOD}"
    echo "  Export Path: ${EXPORT_PATH}"
    echo ""

    if [ -d "${EXPORT_PATH}" ]; then
        echo "Exported Files:"
        ls -lh "${EXPORT_PATH}/" 2>/dev/null || echo "  (path not accessible)"
        echo ""
    fi

    echo "Next Steps:"
    if [ "${HYPER2KVM_ENABLED}" = "true" ]; then
        echo "  1. Check converted files in /var/lib/kvm/images/${VM_ID}/"
        echo "  2. Import to libvirt:"
        echo "     curl -X POST ${DAEMON_URL}/import-to-kvm \\"
        echo "       -H 'Content-Type: application/json' \\"
        echo "       -d '{\"vm_name\": \"${VM_ID}\", \"disk_path\": \"/var/lib/kvm/images/${VM_ID}/disk.qcow2\"}'"
        echo "  3. Start the VM: virsh start ${VM_ID}"
    else
        echo "  1. Manually convert with hyper2kvm:"
        echo "     hyper2kvm convert ${EXPORT_PATH}/*.ovf --output /var/lib/kvm/images/${VM_ID}"
        echo "  2. Import to libvirt"
        echo "  3. Start the VM"
    fi
    echo ""

    log_success "Demo completed!"
}

cleanup() {
    echo ""
    log_info "Cleaning up demo files..."
    rm -rf "${EXPORT_BASE}"
    log_success "Cleanup complete"
}

show_usage() {
    cat <<EOF
HyperSDK → hyper2kvm Workflow Demo

Usage: $0 [OPTIONS]

Options:
  --daemon-url URL    HyperSDK daemon URL (default: http://localhost:8080)
  --provider NAME     Provider name (default: vsphere)
  --no-cleanup        Don't cleanup demo files after completion
  -h, --help          Show this help message

Examples:
  $0
  $0 --daemon-url http://192.168.1.100:8080
  $0 --provider aws --no-cleanup

EOF
}

# Main execution
main() {
    # Parse arguments
    CLEANUP_ENABLED=true

    while [ $# -gt 0 ]; do
        case $1 in
            --daemon-url)
                DAEMON_URL="$2"
                shift 2
                ;;
            --provider)
                PROVIDER="$2"
                shift 2
                ;;
            --no-cleanup)
                CLEANUP_ENABLED=false
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Print banner
    clear
    cat <<EOF
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║    HyperSDK → hyper2kvm Integration Workflow Demo       ║
║                                                          ║
║    This demo will walk through:                         ║
║    1. Checking daemon capabilities                      ║
║    2. Listing available VMs                             ║
║    3. Selecting a VM to export                          ║
║    4. Configuring export options                        ║
║    5. Submitting export job                             ║
║    6. Monitoring job progress                           ║
║    7. Checking hyper2kvm integration                    ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝

EOF

    echo "Configuration:"
    echo "  Daemon URL: ${DAEMON_URL}"
    echo "  Provider: ${PROVIDER}"
    echo ""
    read -p "Press Enter to start the demo..."

    # Run demo steps
    check_dependencies
    check_daemon
    show_capabilities
    list_vms
    select_vm
    configure_export_options
    submit_job
    monitor_job
    check_hyper2kvm
    show_summary

    # Cleanup
    if [ "${CLEANUP_ENABLED}" = true ]; then
        echo ""
        read -p "Cleanup demo files? [Y/n] " CLEANUP_CONFIRM
        if [ "${CLEANUP_CONFIRM}" != "n" ] && [ "${CLEANUP_CONFIRM}" != "N" ]; then
            cleanup
        fi
    fi

    echo ""
    log_success "Thank you for trying the HyperSDK → hyper2kvm demo!"
}

# Run main function
main "$@"
