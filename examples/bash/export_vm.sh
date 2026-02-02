#!/bin/bash
#
# Simple VM Export Script
#
# This script exports a VM using the HyperSDK REST API.
#
# Usage:
#   ./export_vm.sh /datacenter/vm/my-vm /exports
#

set -e

# Configuration
API_URL="${HYPERSDK_API_URL:-http://localhost:8080}"
VM_PATH="${1:-}"
OUTPUT_PATH="${2:-}"
FORMAT="${3:-ova}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Usage
if [ -z "$VM_PATH" ] || [ -z "$OUTPUT_PATH" ]; then
    echo "Usage: $0 <vm-path> <output-path> [format]"
    echo ""
    echo "Example:"
    echo "  $0 /datacenter/vm/my-vm /exports ova"
    echo ""
    echo "Environment variables:"
    echo "  HYPERSDK_API_URL - API URL (default: http://localhost:8080)"
    exit 1
fi

echo -e "${BLUE}🚀 HyperSDK VM Export${NC}"
echo -e "   API URL: $API_URL"
echo -e "   VM Path: $VM_PATH"
echo -e "   Output: $OUTPUT_PATH"
echo -e "   Format: $FORMAT"
echo ""

# Step 1: Check API health
echo -e "${BLUE}📡 Checking API health...${NC}"
if ! HEALTH=$(curl -s -f "$API_URL/health"); then
    echo -e "${RED}❌ API is not reachable at $API_URL${NC}"
    exit 1
fi
echo -e "${GREEN}✅ API is healthy${NC}"
echo ""

# Step 2: Submit export job
echo -e "${BLUE}📤 Submitting export job...${NC}"
RESPONSE=$(curl -s -X POST "$API_URL/jobs/submit" \
    -H "Content-Type: application/json" \
    -d "{
        \"vm_path\": \"$VM_PATH\",
        \"output_path\": \"$OUTPUT_PATH\",
        \"format\": \"$FORMAT\",
        \"compression\": true,
        \"verify\": true
    }")

# Extract job ID
JOB_ID=$(echo "$RESPONSE" | jq -r '.job_ids[0]')

if [ "$JOB_ID" = "null" ] || [ -z "$JOB_ID" ]; then
    echo -e "${RED}❌ Failed to submit job${NC}"
    echo "$RESPONSE" | jq '.'
    exit 1
fi

echo -e "${GREEN}✅ Job submitted: $JOB_ID${NC}"
echo ""

# Step 3: Monitor progress
echo -e "${BLUE}⏳ Monitoring progress...${NC}"
LAST_PERCENT=""

while true; do
    JOB=$(curl -s "$API_URL/jobs/$JOB_ID")
    STATUS=$(echo "$JOB" | jq -r '.status')

    case "$STATUS" in
        "completed")
            echo ""
            echo -e "${GREEN}✅ Export completed!${NC}"
            OVF_PATH=$(echo "$JOB" | jq -r '.result.ovf_path // "N/A"')
            DURATION=$(echo "$JOB" | jq -r '.duration // "N/A"')
            echo -e "   OVF Path: $OVF_PATH"
            echo -e "   Duration: $DURATION"
            exit 0
            ;;

        "failed")
            echo ""
            echo -e "${RED}❌ Export failed!${NC}"
            ERROR=$(echo "$JOB" | jq -r '.error // "Unknown error"')
            echo -e "   Error: $ERROR"
            exit 1
            ;;

        "running")
            PERCENT=$(echo "$JOB" | jq -r '.progress.percent_complete // 0')
            PHASE=$(echo "$JOB" | jq -r '.progress.phase // "unknown"')
            SPEED=$(echo "$JOB" | jq -r '.progress.speed_mbps // 0')

            # Only update if percent changed
            if [ "$PERCENT" != "$LAST_PERCENT" ]; then
                printf "\r${YELLOW}⏳ %s: %.1f%% complete (%.1f MB/s)${NC}" \
                    "$PHASE" "$PERCENT" "$SPEED"
                LAST_PERCENT="$PERCENT"
            fi
            ;;

        *)
            PHASE=$(echo "$JOB" | jq -r '.progress.phase // "waiting"')
            printf "\r${YELLOW}⏳ %s...${NC}" "$PHASE"
            ;;
    esac

    sleep 5
done
