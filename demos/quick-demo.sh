#!/bin/bash
# Quick demo of hypersdk tools
# Usage: ./demos/quick-demo.sh

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Helper function
demo_section() {
    echo
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo
    sleep 1
}

clear

echo -e "${BLUE}"
cat << "EOF"
╔═══════════════════════════════════════╗
║     hypersdk Quick Demo               ║
║     Multi-Cloud VM Export Toolkit     ║
╚═══════════════════════════════════════╝
EOF
echo -e "${NC}"

sleep 2

# Demo 1: Show versions
demo_section "1. Check Tool Versions"
echo "$ ./build/hypervisord --version"
./build/hypervisord --version
sleep 1

echo
echo "$ ./build/hyperctl --version"
./build/hyperctl --version
sleep 1

echo
echo "$ ./build/hyperexport --version 2>&1 | head -1 || echo 'Interactive tool - no version flag'"
sleep 1

# Demo 2: Daemon help
demo_section "2. Hypervisord Daemon Options"
echo "$ ./build/hypervisord --help"
./build/hypervisord --help
sleep 2

# Demo 3: Control CLI help
demo_section "3. Hyperctl Control CLI"
echo "$ ./build/hyperctl --help"
./build/hyperctl --help
sleep 2

# Demo 4: If daemon is running, show status
demo_section "4. Daemon Status (if running)"
echo "$ ./build/hyperctl status"
if ./build/hyperctl status 2>/dev/null; then
    echo
    echo -e "${GREEN}✓ Daemon is running!${NC}"
else
    echo -e "${BLUE}ℹ Daemon not running - start with: ./build/hypervisord${NC}"
fi
sleep 2

# Demo 5: List VMs if daemon is running
demo_section "5. VM Discovery (if daemon running)"
echo "$ ./build/hyperctl list | head -15"
if ./build/hyperctl list 2>/dev/null | head -15; then
    echo
    echo -e "${GREEN}✓ Successfully discovered VMs!${NC}"
else
    echo -e "${BLUE}ℹ Start daemon first: ./build/hypervisord${NC}"
fi
sleep 2

# Completion
echo
echo -e "${GREEN}"
cat << "EOF"
╔═══════════════════════════════════════╗
║     Demo Complete!                    ║
║                                       ║
║  Next Steps:                          ║
║  • Start daemon: ./build/hypervisord  ║
║  • List VMs: ./build/hyperctl list    ║
║  • Export VMs: ./build/hyperexport    ║
╚═══════════════════════════════════════╝
EOF
echo -e "${NC}"
