#!/bin/bash
# Comprehensive Dashboard Endpoint Testing Script
# Tests all endpoints used by the HyperSDK dashboard

API_BASE="http://localhost:8080"
PASS=0
FAIL=0
MISSING=0

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "HyperSDK Dashboard Endpoint Testing"
echo "========================================="
echo ""

test_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=$3
    local description=$4
    local data=$5

    if [ "$method" == "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE$endpoint")
    elif [ "$method" == "POST" ]; then
        if [ -n "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X POST -H "Content-Type: application/json" -d "$data" "$API_BASE$endpoint")
        else
            response=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE$endpoint")
        fi
    fi

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" == "$expected_status" ]; then
        echo -e "${GREEN}✅ PASS${NC} - $method $endpoint ($description)"
        ((PASS++))
        return 0
    elif [ "$http_code" == "404" ]; then
        echo -e "${YELLOW}⚠️  MISSING${NC} - $method $endpoint ($description) - Endpoint not found"
        ((MISSING++))
        return 1
    else
        echo -e "${RED}❌ FAIL${NC} - $method $endpoint ($description) - Expected $expected_status, got $http_code"
        ((FAIL++))
        return 1
    fi
}

echo "Testing Core Endpoints:"
echo "------------------------"
test_endpoint "GET" "/health" "200" "Health check"
test_endpoint "GET" "/status" "200" "Server status"
test_endpoint "GET" "/capabilities" "200" "Server capabilities"

echo ""
echo "Testing Job Management:"
echo "------------------------"
test_endpoint "GET" "/jobs/query?all=true" "200" "Query all jobs (GET)"
test_endpoint "POST" "/jobs/query" "200" "Query jobs (POST)" '{"all":true}'
test_endpoint "POST" "/jobs/submit" "200" "Submit job" '{"vm_path":"/test","output_dir":"/tmp","format":"qcow2"}'
test_endpoint "POST" "/jobs/cancel" "200" "Cancel jobs" '{"job_ids":["test-id"]}'

echo ""
echo "Testing VM Management (VMware):"
echo "------------------------"
test_endpoint "GET" "/vms/list" "200" "List VMware VMs"
test_endpoint "GET" "/vms/info?name=test" "200" "Get VM info"

echo ""
echo "Testing Libvirt Domain Management:"
echo "------------------------"
test_endpoint "GET" "/libvirt/domains" "200" "List libvirt domains"
test_endpoint "GET" "/libvirt/domain?name=test-vm" "200" "Get domain details"
test_endpoint "POST" "/libvirt/domain/start" "200" "Start domain" '{"name":"test-vm"}'
test_endpoint "POST" "/libvirt/domain/shutdown" "200" "Shutdown domain" '{"name":"test-vm"}'
test_endpoint "POST" "/libvirt/domain/reboot" "200" "Reboot domain" '{"name":"test-vm"}'
test_endpoint "POST" "/libvirt/domain/pause" "200" "Pause domain" '{"name":"test-vm"}'
test_endpoint "POST" "/libvirt/domain/resume" "200" "Resume domain" '{"name":"test-vm"}'

echo ""
echo "Testing Console & Display:"
echo "------------------------"
test_endpoint "GET" "/console/info?name=test-vm" "200" "Get console info"
test_endpoint "GET" "/console/vnc?name=test-vm" "200" "VNC console page"
test_endpoint "GET" "/console/serial?name=test-vm" "200" "Serial console page"
test_endpoint "GET" "/console/screenshot?name=test-vm" "200" "Take screenshot"

echo ""
echo "Testing Snapshots:"
echo "------------------------"
test_endpoint "GET" "/libvirt/snapshots?name=test-vm" "200" "List snapshots"
test_endpoint "POST" "/libvirt/snapshot/create" "200" "Create snapshot" '{"name":"test-vm","snapshot_name":"snap1"}'
test_endpoint "POST" "/libvirt/snapshot/revert" "200" "Revert snapshot" '{"name":"test-vm","snapshot_name":"snap1"}'
test_endpoint "POST" "/libvirt/snapshot/delete" "200" "Delete snapshot" '{"name":"test-vm","snapshot_name":"snap1"}'

echo ""
echo "Testing Networks:"
echo "------------------------"
test_endpoint "GET" "/libvirt/networks" "200" "List networks"
test_endpoint "GET" "/libvirt/network?name=default" "200" "Get network info"

echo ""
echo "Testing Volumes:"
echo "------------------------"
test_endpoint "GET" "/libvirt/pools" "200" "List storage pools"
test_endpoint "GET" "/libvirt/volumes?pool=default" "200" "List volumes"

echo ""
echo "Testing ISO Management:"
echo "------------------------"
test_endpoint "GET" "/libvirt/isos/list" "200" "List ISOs"
test_endpoint "POST" "/libvirt/domain/attach-iso" "200" "Attach ISO" '{"vm_name":"test-vm","filename":"test.iso"}'
test_endpoint "POST" "/libvirt/domain/detach-iso" "200" "Detach ISO" '{"vm_name":"test-vm"}'

echo ""
echo "Testing Backups:"
echo "------------------------"
test_endpoint "GET" "/libvirt/backup/list" "200" "List backups"
test_endpoint "POST" "/libvirt/backup/create" "200" "Create backup" '{"vm_name":"test-vm"}'

echo ""
echo "Testing Monitoring:"
echo "------------------------"
test_endpoint "GET" "/libvirt/stats?name=test-vm" "200" "Get domain stats"
test_endpoint "GET" "/libvirt/stats/all" "200" "Get all domain stats"

echo ""
echo "Testing Batch Operations:"
echo "------------------------"
test_endpoint "POST" "/libvirt/batch/start" "200" "Batch start" '{"names":["vm1","vm2"]}'
test_endpoint "POST" "/libvirt/batch/stop" "200" "Batch stop" '{"names":["vm1","vm2"]}'

echo ""
echo "Testing Cloning & Templates:"
echo "------------------------"
test_endpoint "POST" "/libvirt/clone" "200" "Clone domain" '{"name":"source-vm","new_name":"cloned-vm"}'
test_endpoint "GET" "/libvirt/template/list" "200" "List templates"

echo ""
echo "Testing Workflow:"
echo "------------------------"
test_endpoint "POST" "/workflow/convert" "200" "Start conversion workflow" '{"vm_path":"/test","output_dir":"/tmp"}'
test_endpoint "GET" "/workflow/status?job_id=test" "200" "Get workflow status"

echo ""
echo "Testing Job Progress:"
echo "------------------------"
test_endpoint "GET" "/jobs/progress/test-job-id" "200" "Get job progress"
test_endpoint "GET" "/jobs/logs/test-job-id" "200" "Get job logs"
test_endpoint "GET" "/jobs/eta/test-job-id" "200" "Get job ETA"

echo ""
echo "Testing WebSocket:"
echo "------------------------"
test_endpoint "GET" "/ws" "200" "WebSocket endpoint"

echo ""
echo "Testing Authentication:"
echo "------------------------"
test_endpoint "POST" "/api/login" "200" "Login" '{"username":"admin","password":"admin"}'
test_endpoint "POST" "/api/logout" "200" "Logout"

echo ""
echo "Testing Schedules:"
echo "------------------------"
test_endpoint "GET" "/schedules" "200" "List schedules"
test_endpoint "POST" "/schedules" "200" "Create schedule" '{"name":"test-schedule","expression":"0 0 * * *"}'

echo ""
echo "Testing Webhooks:"
echo "------------------------"
test_endpoint "GET" "/webhooks" "200" "List webhooks"
test_endpoint "POST" "/webhooks" "200" "Add webhook" '{"url":"http://example.com/webhook"}'

echo ""
echo "========================================="
echo "Test Summary:"
echo "========================================="
echo -e "${GREEN}Passed:${NC}  $PASS"
echo -e "${RED}Failed:${NC}  $FAIL"
echo -e "${YELLOW}Missing:${NC} $MISSING"
echo "Total:   $((PASS + FAIL + MISSING))"
echo ""

if [ $MISSING -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Some endpoints are missing and need to be implemented${NC}"
fi

if [ $FAIL -gt 0 ]; then
    echo -e "${RED}❌ Some tests failed - check endpoint implementations${NC}"
    exit 1
elif [ $MISSING -gt 0 ]; then
    exit 2
else
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
fi
