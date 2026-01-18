// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

// ConsoleInfo represents console connection information
type ConsoleInfo struct {
	VMName       string `json:"vm_name"`
	VNCDisplay   string `json:"vnc_display"`
	VNCPort      int    `json:"vnc_port"`
	VNCPassword  string `json:"vnc_password,omitempty"`
	VNCURL       string `json:"vnc_url"`
	WebVNCURL    string `json:"webvnc_url"`
	HasVNC       bool   `json:"has_vnc"`
	HasSPICE     bool   `json:"has_spice"`
	HasSerial    bool   `json:"has_serial"`
	SerialDevice string `json:"serial_device,omitempty"`
}

// handleGetConsoleInfo gets comprehensive console information
func (s *Server) handleGetConsoleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vmName := r.URL.Query().Get("name")
	if vmName == "" {
		http.Error(w, "missing name parameter", http.StatusBadRequest)
		return
	}

	info := ConsoleInfo{
		VMName: vmName,
	}

	// Get VNC display
	vncCmd := exec.Command("virsh", "vncdisplay", vmName)
	vncOutput, err := vncCmd.Output()
	if err == nil {
		vncDisplay := strings.TrimSpace(string(vncOutput))
		info.VNCDisplay = vncDisplay
		info.HasVNC = true

		// Parse port from :N format
		if strings.HasPrefix(vncDisplay, ":") {
			var portOffset int
			fmt.Sscanf(vncDisplay, ":%d", &portOffset)
			info.VNCPort = 5900 + portOffset
			info.VNCURL = fmt.Sprintf("vnc://localhost:%d", info.VNCPort)
			info.WebVNCURL = fmt.Sprintf("/console/vnc?name=%s&port=%d", vmName, info.VNCPort)
		}
	}

	// Check for SPICE
	cmd := exec.Command("virsh", "domdisplay", vmName)
	output, err := cmd.Output()
	if err == nil {
		display := strings.TrimSpace(string(output))
		if strings.HasPrefix(display, "spice://") {
			info.HasSPICE = true
		}
	}

	// Check for serial console
	cmd = exec.Command("virsh", "domxml", vmName)
	output, err = cmd.Output()
	if err == nil {
		if strings.Contains(string(output), "<serial type") {
			info.HasSerial = true
			info.SerialDevice = "/dev/pts/X" // Will be determined dynamically
		}
	}

	s.jsonResponse(w, http.StatusOK, info)
}

// handleVNCProxy proxies VNC connections for web viewing
func (s *Server) handleVNCProxy(w http.ResponseWriter, r *http.Request) {
	vmName := r.URL.Query().Get("name")
	if vmName == "" {
		http.Error(w, "missing name parameter", http.StatusBadRequest)
		return
	}

	// Get VNC display
	cmd := exec.Command("virsh", "vncdisplay", vmName)
	output, err := cmd.Output()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to get VNC display: %v", err)
		return
	}

	vncDisplay := strings.TrimSpace(string(output))
	var portOffset int
	fmt.Sscanf(vncDisplay, ":%d", &portOffset)
	vncPort := 5900 + portOffset

	// Return HTML page with noVNC viewer
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - VNC Console</title>
    <meta charset="utf-8">
    <style>
        body {
            margin: 0;
            padding: 0;
            background: #1a1d29;
            font-family: monospace;
            overflow: hidden;
        }
        .toolbar {
            background: #2d3748;
            color: white;
            padding: 10px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .vm-name {
            font-weight: bold;
            color: #f0583a;
        }
        .controls button {
            background: #4a5568;
            color: white;
            border: none;
            padding: 8px 16px;
            margin: 0 5px;
            cursor: pointer;
            border-radius: 4px;
        }
        .controls button:hover {
            background: #f0583a;
        }
        #vnc-container {
            width: 100vw;
            height: calc(100vh - 50px);
            display: flex;
            justify-content: center;
            align-items: center;
        }
        #vnc-canvas {
            border: 2px solid #f0583a;
        }
        .status {
            color: #48bb78;
            padding: 5px 10px;
        }
        .error {
            color: #f56565;
            padding: 5px 10px;
        }
        .info-box {
            background: #2d3748;
            color: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="toolbar">
        <div>
            <span class="vm-name">VM: %s</span>
            <span id="status" class="status">Connecting...</span>
        </div>
        <div class="controls">
            <button onclick="sendCtrlAltDel()">Ctrl+Alt+Del</button>
            <button onclick="toggleFullscreen()">Fullscreen</button>
            <button onclick="reconnect()">Reconnect</button>
            <button onclick="window.close()">Close</button>
        </div>
    </div>

    <div id="vnc-container">
        <div class="info-box">
            <h2>VNC Console</h2>
            <p>VM: <strong>%s</strong></p>
            <p>VNC Port: <strong>%d</strong></p>
            <p>Display: <strong>%s</strong></p>
            <br>
            <p style="color: #f0583a;">
                <strong>Note:</strong> For full VNC access, use a VNC client:<br>
                <code>vncviewer localhost:%d</code><br>
                or<br>
                <code>remote-viewer vnc://localhost:%d</code>
            </p>
            <br>
            <p style="font-size: 12px; color: #a0aec0;">
                To enable web-based VNC viewing, install noVNC:<br>
                <code>git clone https://github.com/novnc/noVNC.git</code><br>
                <code>cd noVNC && ./utils/novnc_proxy --vnc localhost:%d</code>
            </p>
            <br>
            <button onclick="openVNCClient()" style="padding: 12px 24px; font-size: 16px; background: #f0583a; color: white; border: none; border-radius: 6px; cursor: pointer;">
                Open VNC Client
            </button>
        </div>
    </div>

    <script>
        const vmName = '%s';
        const vncPort = %d;
        const vncDisplay = '%s';

        function sendCtrlAltDel() {
            fetch('/libvirt/domain/send-key', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({name: vmName, keys: ['KEY_LEFTCTRL', 'KEY_LEFTALT', 'KEY_DELETE']})
            });
        }

        function toggleFullscreen() {
            if (!document.fullscreenElement) {
                document.documentElement.requestFullscreen();
            } else {
                document.exitFullscreen();
            }
        }

        function reconnect() {
            location.reload();
        }

        function openVNCClient() {
            // Try to open with VNC protocol handler
            window.location = 'vnc://localhost:' + vncPort;

            // Also show instructions
            alert('VNC Connection Details:\n\n' +
                  'Host: localhost\n' +
                  'Port: ' + vncPort + '\n' +
                  'Display: ' + vncDisplay + '\n\n' +
                  'Use a VNC client like:\n' +
                  '- TigerVNC Viewer\n' +
                  '- RealVNC\n' +
                  '- Remmina\n' +
                  '- virt-viewer');
        }

        // Auto-detect VNC availability
        fetch('/libvirt/domain?name=' + vmName)
            .then(r => r.json())
            .then(data => {
                if (data.state === 'running') {
                    document.getElementById('status').textContent = 'VM Running - Port: ' + vncPort;
                    document.getElementById('status').className = 'status';
                } else {
                    document.getElementById('status').textContent = 'VM is ' + data.state;
                    document.getElementById('status').className = 'error';
                }
            });
    </script>
</body>
</html>`, vmName, vmName, vmName, vncPort, vncDisplay, vncPort, vncPort, vncPort, vmName, vncPort, vncDisplay)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleSerialConsole provides access to serial console
func (s *Server) handleSerialConsole(w http.ResponseWriter, r *http.Request) {
	vmName := r.URL.Query().Get("name")
	if vmName == "" {
		http.Error(w, "missing name parameter", http.StatusBadRequest)
		return
	}

	// Return HTML page with terminal emulator
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Serial Console</title>
    <meta charset="utf-8">
    <style>
        body {
            margin: 0;
            padding: 0;
            background: #000;
            font-family: 'Courier New', monospace;
            color: #0f0;
        }
        .toolbar {
            background: #2d3748;
            color: white;
            padding: 10px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .vm-name {
            font-weight: bold;
            color: #f0583a;
        }
        .controls button {
            background: #4a5568;
            color: white;
            border: none;
            padding: 8px 16px;
            margin: 0 5px;
            cursor: pointer;
            border-radius: 4px;
        }
        .controls button:hover {
            background: #f0583a;
        }
        #terminal {
            width: 100vw;
            height: calc(100vh - 50px);
            background: #000;
            color: #0f0;
            padding: 10px;
            overflow-y: auto;
            font-size: 14px;
            white-space: pre-wrap;
        }
        .console-command {
            color: #4a9eff;
            margin: 20px 0;
            padding: 15px;
            background: #1a1d29;
            border-left: 4px solid #f0583a;
            border-radius: 4px;
        }
    </style>
</head>
<body>
    <div class="toolbar">
        <div>
            <span class="vm-name">VM: %s - Serial Console</span>
        </div>
        <div class="controls">
            <button onclick="clearTerminal()">Clear</button>
            <button onclick="window.close()">Close</button>
        </div>
    </div>

    <div id="terminal">
        <div class="console-command">
            <strong>Serial Console Access</strong><br><br>
            To connect to the serial console, use one of these methods:<br><br>

            <strong>Method 1: virsh console</strong><br>
            <code>$ virsh console %s</code><br><br>

            <strong>Method 2: Direct PTY connection</strong><br>
            First, find the PTY device:<br>
            <code>$ virsh dumpxml %s | grep "console type"</code><br>
            Then connect:<br>
            <code>$ screen /dev/pts/X</code><br><br>

            <strong>Method 3: Via SSH to host</strong><br>
            <code>$ ssh -t user@host "virsh console %s"</code><br><br>

            <strong>Note:</strong> The VM must have a serial console configured.<br>
            Check VM XML: <code>virsh dumpxml %s | grep serial</code><br><br>

            <div style="margin-top: 20px; padding-top: 20px; border-top: 1px solid #333;">
                <strong>Configure Serial Console:</strong><br>
                If not configured, you can enable it by editing the VM XML:<br>
                <code>$ virsh edit %s</code><br><br>
                Add this inside &lt;devices&gt;:<br>
                <pre style="background: #000; padding: 10px; margin: 10px 0;">
&lt;serial type='pty'&gt;
  &lt;target port='0'/&gt;
&lt;/serial&gt;
&lt;console type='pty'&gt;
  &lt;target type='serial' port='0'/&gt;
&lt;/console&gt;</pre>
            </div>
        </div>

        <div id="output"></div>
    </div>

    <script>
        function clearTerminal() {
            document.getElementById('output').innerHTML = '';
        }

        // Simulate connecting
        setTimeout(() => {
            const output = document.getElementById('output');
            output.innerHTML = '<span style="color: #48bb78;">[INFO]</span> Attempting to connect to serial console...\n';

            // Try to get actual serial device
            fetch('/libvirt/console/serial?name=%s')
                .then(r => r.json())
                .then(data => {
                    if (data.device) {
                        output.innerHTML += '<span style="color: #48bb78;">[INFO]</span> Serial device: ' + data.device + '\n';
                        output.innerHTML += '<span style="color: #f0583a;">[NOTE]</span> Connect using: virsh console %s\n';
                    } else {
                        output.innerHTML += '<span style="color: #f56565;">[ERROR]</span> No serial console configured\n';
                        output.innerHTML += '<span style="color: #f0583a;">[NOTE]</span> See instructions above to configure\n';
                    }
                })
                .catch(e => {
                    output.innerHTML += '<span style="color: #f56565;">[ERROR]</span> Failed to connect: ' + e + '\n';
                });
        }, 500);
    </script>
</body>
</html>`, vmName, vmName, vmName, vmName, vmName, vmName, vmName, vmName, vmName)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleGetSerialDevice gets the serial console device path
func (s *Server) handleGetSerialDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vmName := r.URL.Query().Get("name")
	if vmName == "" {
		http.Error(w, "missing name parameter", http.StatusBadRequest)
		return
	}

	// Get VM XML and extract serial device
	cmd := exec.Command("virsh", "dumpxml", vmName)
	output, err := cmd.Output()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to get VM XML: %v", err)
		return
	}

	xmlStr := string(output)
	device := ""

	// Parse for PTY device
	if strings.Contains(xmlStr, "<serial type='pty'>") {
		lines := strings.Split(xmlStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "<source path=") {
				// Extract path from: <source path='/dev/pts/X'/>
				start := strings.Index(line, "path='")
				if start >= 0 {
					start += 6
					end := strings.Index(line[start:], "'")
					if end >= 0 {
						device = line[start : start+end]
						break
					}
				}
			}
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"vm_name": vmName,
		"device":  device,
		"has_serial": device != "",
	})
}

// handleScreenshot takes a VM screenshot
func (s *Server) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vmName := r.URL.Query().Get("name")
	if vmName == "" {
		http.Error(w, "missing name parameter", http.StatusBadRequest)
		return
	}

	// Take screenshot using virsh screenshot
	cmd := exec.Command("virsh", "screenshot", vmName, "/tmp/"+vmName+"-screenshot.ppm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to take screenshot: %s", string(output))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"message": "Screenshot saved",
		"path": "/tmp/" + vmName + "-screenshot.ppm",
		"note": "Convert with: convert screenshot.ppm screenshot.png",
	})
}

// handleSendKeys sends key combination to VM
func (s *Server) handleSendKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string   `json:"name"`
		Keys []string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Send keys using virsh send-key
	args := []string{"send-key", req.Name}
	args = append(args, req.Keys...)

	cmd := exec.Command("virsh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to send keys: %s", string(output))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "success",
		"message": "Keys sent successfully",
	})
}
