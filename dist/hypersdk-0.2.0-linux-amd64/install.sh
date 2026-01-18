#!/bin/bash
set -e

echo "Installing HyperSDK..."

# Create user
if ! id -u hypersdk >/dev/null 2>&1; then
    sudo useradd -r -s /bin/false hypersdk
fi

# Create directories
sudo mkdir -p /opt/hypersdk/bin
sudo mkdir -p /etc/hypersdk
sudo mkdir -p /var/log/hypersdk
sudo mkdir -p /var/lib/hypersdk

# Copy binaries from bin/ directory
sudo cp bin/* /opt/hypersdk/bin/
sudo chmod +x /opt/hypersdk/bin/*

# Copy config from docs/ directory
if [ ! -f /etc/hypersdk/config.yaml ]; then
    sudo cp docs/config.example.yaml /etc/hypersdk/config.yaml
    echo "Created /etc/hypersdk/config.yaml - please edit with your settings"
fi

# Install systemd service from docs/ directory
sudo cp docs/hypervisord.service /etc/systemd/system/
sudo systemctl daemon-reload

# Set permissions
sudo chown -R hypersdk:hypersdk /opt/hypersdk
sudo chown -R hypersdk:hypersdk /var/log/hypersdk
sudo chown -R hypersdk:hypersdk /var/lib/hypersdk

echo ""
echo "Installation complete!"
echo ""
echo "Next steps:"
echo "1. Edit configuration: sudo nano /etc/hypersdk/config.yaml"
echo "2. Enable service: sudo systemctl enable hypervisord"
echo "3. Start service: sudo systemctl start hypervisord"
echo "4. Check status: sudo systemctl status hypervisord"
echo ""
