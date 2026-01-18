# hypersdk Demos

This directory contains demo scripts and recordings showing how to use hypersdk tools.

## Recording Your Own Demos

### Option 1: Using asciinema (Recommended for CLI tools)

```bash
# Install asciinema
sudo dnf install asciinema

# Record a demo
asciinema rec demo-hyperctl.cast

# Upload to asciinema.org
asciinema upload demo-hyperctl.cast

# Or convert to GIF using agg
cargo install --git https://github.com/asciinema/agg
agg demo-hyperctl.cast demo-hyperctl.gif
```

### Option 2: Using SimpleScreenRecorder (For full GUI demos)

```bash
# Install SimpleScreenRecorder
sudo dnf install simplescreenrecorder

# Record your screen
# File > Output > Output file: demo.mp4
# Start recording, demo the tools, stop recording
```

### Option 3: Upload Video Directly to GitHub

GitHub supports direct video uploads in README.md:

1. Record video (MP4, MOV, WebM)
2. Drag and drop into GitHub issue/PR/README editor
3. GitHub will upload and provide a link
4. Copy the link into your markdown

## Demo Scripts

### Demo 1: hypervisord Daemon

```bash
# Show version
./build/hypervisord --version

# Show help
./build/hypervisord --help

# Start daemon
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1

./build/hypervisord
# Shows: Banner, API endpoints, waiting for jobs
```

**What to show:**
- Clean startup banner
- API endpoint table
- Daemon waiting for jobs
- Health check endpoint responding

### Demo 2: hyperctl - VM Discovery

```bash
# Check daemon status
./build/hyperctl status

# List all VMs
./build/hyperctl list

# Filter VMs
./build/hyperctl list -filter rhel

# Get JSON output
./build/hyperctl list -json | jq '.[0]'
```

**What to show:**
- Beautiful table output
- VM metadata (power state, CPUs, memory, storage)
- Filtering capabilities
- JSON output for automation

### Demo 3: hyperctl - Job Management

```bash
# Submit a job
./build/hyperctl submit -vm "/datacenter/vm/test-vm" -output "/tmp/export"

# Query all jobs
./build/hyperctl query -all

# Query specific job
./build/hyperctl query -id <job-id>

# Cancel a job
./build/hyperctl cancel -id <job-id>
```

**What to show:**
- Job submission
- Real-time progress tracking
- Job status queries
- Beautiful UI with spinners and colors

### Demo 4: hyperexport - Interactive Export

```bash
# Launch interactive mode
./build/hyperexport

# What happens:
# 1. Beautiful HYPER2KVM banner
# 2. Connects to vSphere
# 3. Discovers VMs
# 4. Interactive selection
# 5. VM info display
# 6. Export with progress bars
```

**What to show:**
- Animated banner and UI
- VM discovery spinner
- Interactive VM selection
- Real-time progress tracking
- Successful completion

## Sample Recordings

### Quick Demo (30 seconds)

```bash
#!/bin/bash
# demos/quick-demo.sh

echo "=== hypersdk Quick Demo ==="
echo

echo "1. Check daemon status"
./build/hyperctl status
sleep 2

echo
echo "2. List VMs"
./build/hyperctl list | head -10
sleep 2

echo
echo "3. Show version"
./build/hypervisord --version
./build/hyperctl --version
```

### Full Workflow Demo (2-3 minutes)

```bash
#!/bin/bash
# demos/full-workflow-demo.sh

echo "=== hypersdk Full Workflow Demo ==="

# Step 1: Start daemon
echo "Starting hypervisord daemon..."
./build/hypervisord &
DAEMON_PID=$!
sleep 3

# Step 2: Check status
echo "Checking daemon status..."
./build/hyperctl status

# Step 3: List VMs
echo "Discovering VMs..."
./build/hyperctl list | head -20

# Step 4: Submit export job
echo "Submitting export job..."
./build/hyperctl submit \
  -vm "/datacenter/vm/test-vm" \
  -output "/tmp/demo-export"

# Step 5: Monitor job
echo "Monitoring job progress..."
./build/hyperctl query -all

# Cleanup
kill $DAEMON_PID
```

## Tips for Great Demos

1. **Terminal Setup:**
   - Use a clean, large terminal (100x30 minimum)
   - Good color scheme (dark background recommended)
   - Clear font (Fira Code, JetBrains Mono, etc.)

2. **Recording:**
   - Start with `clear` to clean the screen
   - Type commands slowly and deliberately
   - Add `sleep` commands between steps
   - Show both successes and how errors are handled

3. **Content:**
   - Keep it under 2-3 minutes
   - Show the most impressive features first
   - Demonstrate real value (not just --help)
   - End with a successful completion

4. **Quality:**
   - 1920x1080 resolution preferred
   - 60 FPS for smooth animations
   - Include audio narration if possible
   - Add text overlays explaining what's happening

## Embedding in README

### Asciinema

```markdown
[![asciicast](https://asciinema.org/a/RECORDING_ID.svg)](https://asciinema.org/a/RECORDING_ID)
```

### GIF

```markdown
![Demo](demos/hyperctl-demo.gif)
```

### Video (GitHub native)

```markdown
https://user-images.githubusercontent.com/...
```

Or just drag and drop MP4 files directly into GitHub's markdown editor!

## Pre-recorded Demos

- `hypervisord-startup.cast` - Daemon startup and API endpoints
- `hyperctl-discovery.cast` - VM discovery and listing
- `hyperctl-jobs.cast` - Job management workflow
- `hyperexport-interactive.cast` - Full interactive export

Use `asciinema play <file>.cast` to view them locally.
