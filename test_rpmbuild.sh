#!/bin/bash
# SPDX-License-Identifier: LGPL-3.0-or-later
# RPM Build Test Script for hypersdk
#
# This script tests the RPM build process locally before pushing to CI/CD

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== hypersdk RPM Build Test ===${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check dependencies
echo -e "${YELLOW}Checking dependencies...${NC}"
MISSING_DEPS=()

if ! command_exists rpmbuild; then
    MISSING_DEPS+=("rpmdevtools")
fi

if ! command_exists go; then
    MISSING_DEPS+=("golang")
fi

if ! command_exists git; then
    MISSING_DEPS+=("git")
fi

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
    echo -e "${RED}Error: Missing dependencies:${NC} ${MISSING_DEPS[*]}"
    echo "Install them with: sudo dnf install ${MISSING_DEPS[*]}"
    exit 1
fi

echo -e "${GREEN}✓ All dependencies found${NC}"
echo ""

# Get version from spec file
VERSION=$(grep "^Version:" hypersdk.spec | awk '{print $2}')
echo -e "${YELLOW}Building RPM for version:${NC} $VERSION"
echo ""

# Create RPM build structure
echo -e "${YELLOW}Setting up RPM build environment...${NC}"
RPMBUILD_DIR="$HOME/rpmbuild"
mkdir -p "$RPMBUILD_DIR"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Create source tarball
echo -e "${YELLOW}Creating source tarball...${NC}"
TARBALL="$RPMBUILD_DIR/SOURCES/hypersdk-$VERSION.tar.gz"

# Create a clean export
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

git archive --format=tar --prefix="hypersdk-$VERSION/" HEAD | gzip > "$TARBALL"

if [ ! -f "$TARBALL" ]; then
    echo -e "${RED}Error: Failed to create source tarball${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Source tarball created:${NC} $TARBALL"
echo ""

# Copy spec file
cp hypersdk.spec "$RPMBUILD_DIR/SPECS/"

# Run rpmbuild
echo -e "${YELLOW}Running rpmbuild...${NC}"
echo ""

if rpmbuild -ba "$RPMBUILD_DIR/SPECS/hypersdk.spec"; then
    echo ""
    echo -e "${GREEN}✓ RPM build successful!${NC}"
    echo ""
    echo "Built packages:"
    find "$RPMBUILD_DIR/RPMS" -name "*.rpm" -exec ls -lh {} \;
    echo ""
    echo "Source RPM:"
    find "$RPMBUILD_DIR/SRPMS" -name "*.rpm" -exec ls -lh {} \;
    echo ""

    # Optional: Run rpmlint if available
    if command_exists rpmlint; then
        echo -e "${YELLOW}Running rpmlint...${NC}"
        RPM_FILE=$(find "$RPMBUILD_DIR/RPMS" -name "hypersdk-*.rpm" | head -1)
        if [ -n "$RPM_FILE" ]; then
            rpmlint "$RPM_FILE" || echo -e "${YELLOW}Note: rpmlint warnings found (review above)${NC}"
        fi
        echo ""
    fi

    # Test package metadata
    echo -e "${YELLOW}Package information:${NC}"
    RPM_FILE=$(find "$RPMBUILD_DIR/RPMS" -name "hypersdk-*.rpm" | head -1)
    if [ -n "$RPM_FILE" ]; then
        rpm -qip "$RPM_FILE"
        echo ""
        echo -e "${YELLOW}Package contents:${NC}"
        rpm -qlp "$RPM_FILE"
    fi

    echo ""
    echo -e "${GREEN}=== RPM Build Test PASSED ===${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}=== RPM Build Test FAILED ===${NC}"
    exit 1
fi
