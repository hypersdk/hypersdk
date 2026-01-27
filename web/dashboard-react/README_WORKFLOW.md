# HyperSDK Web Dashboard - Export Workflow

## Overview

The HyperSDK web dashboard now includes a comprehensive **Login → Auto-Discover → Export** workflow that works with all 9 supported cloud providers.

## New Features

### 1. Provider Manager
**File:** `src/components/ProviderManager.tsx`

Centralized provider connection management with:
- Add/configure multiple cloud providers
- Test connections before use
- Visual status indicators (connected/disconnected)
- Provider-specific configuration forms
- Browse VMs directly from connected providers

**Supported Providers:**
- ☁️ VMware vSphere/ESXi
- 🌐 Amazon AWS EC2
- 🔷 Microsoft Azure VMs
- 🔶 Google Cloud Platform
- 💻 Microsoft Hyper-V
- 🟧 Oracle Cloud Infrastructure (OCI)
- 🌀 OpenStack Nova
- 🟠 Alibaba Cloud ECS
- 🔧 Proxmox VE

### 2. VM Browser
**File:** `src/components/VMBrowser.tsx`

Auto-discovery and browsing of virtual machines with:
- **Auto-discovery on mount:** Automatically lists VMs when provider is selected
- **Search & filter:** Find VMs by name, ID, or power state
- **Sortable columns:** Sort by name, CPU, memory, status
- **Real-time refresh:** Update VM list with one click
- **Visual indicators:** Color-coded power states
- **Selection interface:** Click to select VM for export

**Features:**
```typescript
<VMBrowser
  provider="vsphere"
  onVMSelect={(vm) => console.log('Selected:', vm)}
  autoDiscoverOnMount={true}
/>
```

### 3. Export Workflow
**File:** `src/components/ExportWorkflow.tsx`

Guided 3-step workflow:

#### Step 1: Select Provider
- View all configured providers
- Add new provider connections
- Test connectivity
- Select provider to browse VMs

#### Step 2: Discover & Select VM
- Automatically discover VMs from selected provider
- Filter by status, search by name
- View VM details (CPU, memory, OS, IP)
- Select VM for export

#### Step 3: Configure Export
- Pre-filled with selected VM details
- Choose export format (OVA, OVF, VMDK, etc.)
- Enable compression
- Configure hyper2kvm integration
- Submit export job

**Progress Indicator:**
```
[1] Select Provider → [2] Select VM → [3] Configure Export
```

**Breadcrumb Navigation:**
```
🔌 Providers / 📂 VSPHERE VMs / ⚙️ Export vm-name
```

### 4. Enhanced App Navigation
**File:** `src/App.tsx`

New top navigation bar with:
- 📊 Dashboard - System overview and metrics
- 📤 Export Workflow - Guided VM export
- 📋 Jobs - Job history and management
- ⚙️ Workflows - Workflow automation
- 🚪 Logout - End session

**Authentication:**
- Login screen with credentials
- Token-based session management
- localStorage persistence
- Automatic logout on session end

## API Endpoints Used

### Provider Management
```
GET  /api/providers/list          - List configured providers
POST /api/providers/add            - Add new provider
POST /api/providers/test           - Test provider connection
```

### VM Discovery
```
POST /api/vms/list                 - List VMs from provider
GET  /api/vms/info?identifier=X    - Get VM details
```

### Job Submission
```
POST /api/jobs/submit              - Submit export job
GET  /api/jobs/query?job_id=X      - Query job status
GET  /api/jobs/progress/X          - Get job progress
```

## Usage

### Starting the Dashboard

```bash
cd web/dashboard-react
npm install
npm run dev
```

The dashboard will be available at `http://localhost:5173`

### Production Build

```bash
npm run build
# Output: daemon/dashboard/static-react/
```

## Workflow Example

### 1. Login
```
User enters credentials → App authenticates → Dashboard loads
```

### 2. Navigate to Export Workflow
```
Click "📤 Export Workflow" in navigation bar
```

### 3. Select Provider
```
View configured providers → Click "Add Provider" (if needed)
→ Fill provider credentials → Test connection
→ Click "Browse VMs" on connected provider
```

### 4. Discover VMs
```
VMs auto-discovered on load → Use search/filter to find VM
→ Click on VM row → Click "Select for Export"
```

### 5. Configure Export
```
Review pre-filled VM details → Choose export format
→ Enable compression (optional) → Enable hyper2kvm integration (optional)
→ Click "Submit Job"
```

### 6. Monitor Progress
```
Navigate to "📋 Jobs" → View job status and progress
→ Real-time updates via WebSocket
```

## Component API

### ProviderManager

```typescript
interface ProviderManagerProps {
  onProviderSelect?: (provider: string) => void;
}

// Usage
<ProviderManager
  onProviderSelect={(provider) => {
    console.log('Selected provider:', provider);
  }}
/>
```

### VMBrowser

```typescript
interface VMBrowserProps {
  provider: string;                    // Provider ID (vsphere, aws, etc.)
  onVMSelect?: (vm: VM) => void;       // Callback when VM is selected
  autoDiscoverOnMount?: boolean;        // Auto-discover on component mount
}

// Usage
<VMBrowser
  provider="vsphere"
  onVMSelect={(vm) => setSelectedVM(vm)}
  autoDiscoverOnMount={true}
/>
```

### ExportWorkflow

```typescript
// No props required - fully self-contained workflow
<ExportWorkflow />
```

### JobSubmissionForm

```typescript
interface JobSubmissionFormProps {
  onSubmit?: (jobData: any) => Promise<void>;
  initialProvider?: string;
  initialVMIdentifier?: string;
  initialVMName?: string;
}

// Usage
<JobSubmissionForm
  initialProvider="vsphere"
  initialVMIdentifier="vm-123"
  initialVMName="web-server-01"
/>
```

## Styling

All components use inline styles with consistent theme:
- **Primary Color:** `#f0583a` (orange)
- **Background:** `#f0f2f7` (light gray)
- **Text:** `#222324` (dark gray)
- **Success:** `#4caf50` (green)
- **Error:** `#f44336` (red)
- **Info:** `#2196f3` (blue)

No external CSS files required.

## State Management

### App-Level State
- `isAuthenticated` - User login status
- `authToken` - Session token
- `currentView` - Active navigation tab

### Workflow State
- `currentStep` - Active workflow step (providers/vms/export)
- `selectedProvider` - Currently selected provider
- `selectedVM` - Currently selected VM

### Form State
- `formData` - Job submission form values
- `providerConfig` - Provider connection settings

## Backend Integration

### Provider Configuration Storage

The backend should implement provider CRUD operations:

```go
// GET /api/providers/list
func ListProviders(w http.ResponseWriter, r *http.Request) {
    // Return list of configured providers
    providers := []ProviderConfig{
        {
            Provider: "vsphere",
            Name: "Production vCenter",
            Enabled: true,
            Connected: true,
            Config: map[string]string{
                "host": "vcenter.example.com",
                // Don't return passwords
            },
        },
    }
    json.NewEncoder(w).Encode(providers)
}

// POST /api/providers/add
func AddProvider(w http.ResponseWriter, r *http.Request) {
    // Parse request, validate, store provider config
    // Encrypt sensitive fields (passwords, keys)
}

// POST /api/providers/test
func TestProvider(w http.ResponseWriter, r *http.Request) {
    // Test provider connection
    // Return success/failure with error message
}
```

### VM Discovery

```go
// POST /api/vms/list
func ListVMs(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Provider string            `json:"provider"`
        Filter   map[string]string `json:"filter"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Use provider to discover VMs
    provider := GetProvider(req.Provider)
    vms, err := provider.ListVMs(context.Background(), req.Filter)

    json.NewEncoder(w).Encode(vms)
}
```

## Security Considerations

### Authentication
- Implement proper session token validation
- Use secure cookies with HttpOnly flag
- Implement CSRF protection
- Add rate limiting on login endpoint

### Provider Credentials
- Encrypt sensitive fields in database
- Never return passwords in API responses
- Use environment variables for default credentials
- Implement RBAC for provider access

### API Security
- Validate all input parameters
- Sanitize user-provided data
- Implement request size limits
- Add audit logging for sensitive operations

## Testing

### Unit Tests
```bash
npm run test
```

### E2E Testing
```typescript
describe('Export Workflow', () => {
  it('should complete full workflow', async () => {
    // 1. Login
    await login('user', 'pass');

    // 2. Navigate to export
    await click('Export Workflow');

    // 3. Select provider
    await click('vSphere Provider');

    // 4. Select VM
    await waitForVMs();
    await click('vm-001');

    // 5. Submit export
    await fillForm({ format: 'ova' });
    await click('Submit Job');

    // Verify job created
    expect(jobId).toBeDefined();
  });
});
```

## Troubleshooting

### VMs Not Discovered
**Issue:** VMBrowser shows "No virtual machines found"

**Solutions:**
1. Check provider connection status
2. Test provider connectivity in ProviderManager
3. Verify API endpoint `/api/vms/list` is implemented
4. Check browser console for errors
5. Verify provider credentials are correct

### Provider Connection Fails
**Issue:** "Connection test failed" error

**Solutions:**
1. Verify provider credentials
2. Check network connectivity to provider
3. Disable SSL verification for self-signed certs (vSphere)
4. Check firewall rules
5. Review backend logs

### Export Job Fails
**Issue:** Job status shows "failed"

**Solutions:**
1. Check job logs via API: `GET /api/jobs/logs/{job_id}`
2. Verify disk space on export destination
3. Check provider permissions
4. Review daemon logs

## Future Enhancements

### Planned Features
- [ ] Bulk VM selection for batch exports
- [ ] Scheduled export creation from workflow
- [ ] VM console viewer integration
- [ ] Advanced filters (by tag, resource pool, etc.)
- [ ] Export templates/presets
- [ ] Progress visualization in workflow
- [ ] Dark mode support
- [ ] Mobile-responsive design

### Provider-Specific Features
- [ ] vSphere: Folder/datacenter tree view
- [ ] AWS: Region selector
- [ ] Azure: Subscription/resource group selector
- [ ] GCP: Project/zone selector

## Contributing

To add support for new cloud providers:

1. Add provider to `AVAILABLE_PROVIDERS` in `ProviderManager.tsx`
2. Add configuration fields in `getProviderFields()`
3. Add provider to `CloudProvider` type
4. Update backend provider registry
5. Test connection and VM discovery
6. Update documentation

## License

See main project LICENSE file.

---

**Last Updated:** 2026-01-29

**Component Version:** 2.0.0 (with workflow integration)

**Requires:** HyperSDK daemon v1.0.0+
