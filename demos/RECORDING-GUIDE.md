# Video Recording Guide

## Quick Start - Record & Upload to GitHub

### Method 1: SimpleScreenRecorder (Easiest for Full Demos)

```bash
# Install
sudo dnf install simplescreenrecorder

# Record:
# 1. Open SimpleScreenRecorder
# 2. Select area to record (your terminal)
# 3. Output: ~/Videos/hypersdk-demo.mp4
# 4. Click "Start Recording"
# 5. Run: ./demos/quick-demo.sh
# 6. Click "Stop Recording"
```

### Method 2: OBS Studio (Professional Quality)

```bash
# Install
sudo dnf install obs-studio

# Configure:
# 1. Add Source > Window Capture > Select Terminal
# 2. Settings > Output > Recording > MP4 format
# 3. Start Recording
# 4. Run demos
# 5. Stop Recording
```

### Method 3: asciinema (Best for CLI Tools)

```bash
# Record
asciinema rec demos/hyperctl-demo.cast

# Upload to asciinema.org
asciinema upload demos/hyperctl-demo.cast
# You'll get: https://asciinema.org/a/XXXXXXX

# Or convert to GIF
cargo install --git https://github.com/asciinema/agg
agg demos/hyperctl-demo.cast demos/hyperctl-demo.gif
```

## Upload to GitHub

### Option A: GitHub Releases (Recommended)

```bash
# 1. Create a release
gh release create v0.0.1-demo

# 2. Upload video
gh release upload v0.0.1-demo demos/hypersdk-demo.mp4

# 3. Get URL and add to README
```

### Option B: Drag & Drop (Easiest)

1. Go to GitHub repository
2. Click "Edit" on README.md
3. Drag and drop your MP4 file into the editor
4. GitHub will upload and provide markdown like:
   ```
   https://github.com/user-attachments/assets/...
   ```
5. Copy the URL and save

### Option C: Git LFS (For Large Files)

```bash
# Install Git LFS
sudo dnf install git-lfs
git lfs install

# Track video files
git lfs track "demos/*.mp4"
git lfs track "demos/*.gif"

# Add and commit
git add .gitattributes demos/hypersdk-demo.mp4
git commit -m "Add demo video"
git push
```

## What to Record

### Demo 1: Quick Overview (30-60 seconds)

```bash
./demos/quick-demo.sh
```

**Shows:**
- Tool versions
- Daemon help
- Control CLI capabilities
- Beautiful terminal UI

### Demo 2: hypervisord Daemon (60 seconds)

**Script:**
```bash
clear
echo "Starting hypervisord daemon..."
./build/hypervisord
# Let it show banner and API endpoints
# Press Ctrl+C after 10 seconds
```

**Shows:**
- Clean startup
- Beautiful banner
- API endpoint table
- Ready state

### Demo 3: hyperctl Discovery (90 seconds)

**Script:**
```bash
# Ensure daemon is running first!
clear

echo "Checking daemon status..."
./build/hyperctl status

echo "Discovering VMs..."
./build/hyperctl list | head -20

echo "Filtering VMs..."
./build/hyperctl list -filter rhel

echo "JSON output..."
./build/hyperctl list -json | jq '.[0]'
```

**Shows:**
- Daemon connectivity
- VM discovery
- Beautiful tables
- Filtering
- JSON output

### Demo 4: hyperexport Interactive (2 minutes)

**Script:**
```bash
clear
./build/hyperexport
# Let it show:
# - Banner animation
# - Connection spinner
# - VM discovery
# - Interactive selection (just demo, don't complete)
# Press Ctrl+C to exit
```

**Shows:**
- Beautiful ASCII art banner
- Connection progress
- VM list with real-time discovery
- Interactive terminal UI

## Recording Tips

### Terminal Setup

```bash
# Set good size
resize -s 30 100

# Use clean prompt
export PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '

# Clear screen
clear
```

### Recording Quality

- **Resolution:** 1920x1080 minimum
- **Frame Rate:** 30 FPS minimum (60 FPS better)
- **Duration:** 30 seconds - 3 minutes per demo
- **Format:** MP4 (H.264) for best GitHub compatibility

### Before Recording

```bash
# Build everything
make clean && make build

# Test your scripts
./demos/quick-demo.sh

# Ensure daemon credentials are set
export GOVC_URL='https://your-vcenter/sdk'
export GOVC_USERNAME='admin@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1
```

## After Recording

### Trim/Edit (Optional)

```bash
# Using ffmpeg
ffmpeg -i input.mp4 -ss 00:00:02 -to 00:01:30 -c copy output.mp4

# Remove audio (if needed)
ffmpeg -i input.mp4 -an -c:v copy output-silent.mp4
```

### Optimize Size

```bash
# Compress video
ffmpeg -i input.mp4 -vcodec h264 -acodec aac \
  -b:v 1M -b:a 128k output-compressed.mp4

# Convert to GIF (smaller but lower quality)
ffmpeg -i input.mp4 -vf "fps=10,scale=800:-1:flags=lanczos" \
  -c:v gif output.gif
```

### Add to README

```markdown
## 📹 Video Demos

### Quick Start Demo
<video src="https://user-attachments.githubusercontent.com/.../demo.mp4"></video>

### hypervisord Daemon
![Daemon Demo](demos/hypervisord-demo.gif)

### hyperctl Control CLI
[![asciicast](https://asciinema.org/a/XXXXXXX.svg)](https://asciinema.org/a/XXXXXXX)
```

## Example asciinema Recording

```bash
# Start recording
asciinema rec demo.cast

# Run commands
./build/hypervisord --version
./build/hyperctl --version
./build/hyperctl status

# Stop recording (Ctrl+D)

# Play it back
asciinema play demo.cast

# Upload
asciinema upload demo.cast
```

## Need Help?

- asciinema docs: https://docs.asciinema.org/
- GitHub video docs: https://github.blog/changelog/2021-05-13-video-uploads-available-for-github-enterprise-cloud-and-server/
- OBS Studio guide: https://obsproject.com/wiki/
