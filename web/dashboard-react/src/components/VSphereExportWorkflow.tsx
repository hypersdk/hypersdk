import React, { useState, useEffect } from 'react';

interface VM {
  id: string;
  name: string;
  power_state: string;
  cpu_count?: number;
  memory_mb?: number;
  os?: string;
  ip_address?: string;
  datacenter?: string;
  cluster?: string;
}

type WorkflowStep = 'login' | 'discover' | 'export';

const VSphereExportWorkflow: React.FC = () => {
  const [currentStep, setCurrentStep] = useState<WorkflowStep>('login');
  const [vSphereConfig, setVSphereConfig] = useState({
    vcenter: '',
    datacenter: '',
    username: '',
    password: '',
    insecure: true,
  });
  const [vms, setVms] = useState<VM[]>([]);
  const [selectedVM, setSelectedVM] = useState<VM | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [rememberVSphere, setRememberVSphere] = useState(false);

  // Load saved vSphere credentials on mount
  useEffect(() => {
    const savedVCenter = localStorage.getItem('vsphere_vcenter');
    const savedDatacenter = localStorage.getItem('vsphere_datacenter');
    const savedUsername = localStorage.getItem('vsphere_username');
    const savedPassword = localStorage.getItem('vsphere_password');
    const savedInsecure = localStorage.getItem('vsphere_insecure');

    if (savedVCenter && savedUsername && savedPassword) {
      setVSphereConfig({
        vcenter: savedVCenter,
        datacenter: savedDatacenter || '',
        username: savedUsername,
        password: savedPassword,
        insecure: savedInsecure === 'true',
      });
      setRememberVSphere(true);
    }
  }, []);

  // Export options
  const [exportOptions, setExportOptions] = useState({
    jobName: '',
    outputDir: '/tmp/exports',
    format: 'ova',
    compress: false,
    enablePipeline: false,
  });

  // Step 1: vSphere Login
  const handleVSphereLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      // Trim all inputs to remove any whitespace (including non-breaking spaces from copy-paste)
      const cleanedConfig = {
        vcenter: vSphereConfig.vcenter.trim().replace(/\s+/g, ''),
        datacenter: vSphereConfig.datacenter.trim(),
        username: vSphereConfig.username.trim(),
        password: vSphereConfig.password.trim(),
        insecure: vSphereConfig.insecure,
      };

      // Call API to discover VMs from vSphere (backend expects GET with query params)
      const params = new URLSearchParams({
        server: cleanedConfig.vcenter,
        username: cleanedConfig.username,
        password: cleanedConfig.password,
        insecure: cleanedConfig.insecure.toString(),
      });

      const response = await fetch(`/vms/list?${params}`, {
        method: 'GET',
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('API Error:', response.status, errorText);
        throw new Error(errorText || 'Failed to connect to vSphere. Please check your credentials.');
      }

      const discoveredVMs = await response.json();
      console.log('Discovered VMs Response:', discoveredVMs);

      // Backend returns { vms: [...], total: N, timestamp: ... }
      const vmList = discoveredVMs.vms || discoveredVMs.VMs || [];
      console.log('VM List:', vmList);
      setVms(Array.isArray(vmList) ? vmList : []);

      // Update state with cleaned values
      setVSphereConfig(cleanedConfig);

      // Save vSphere credentials if "Remember" is checked
      if (rememberVSphere) {
        localStorage.setItem('vsphere_vcenter', cleanedConfig.vcenter);
        localStorage.setItem('vsphere_datacenter', cleanedConfig.datacenter);
        localStorage.setItem('vsphere_username', cleanedConfig.username);
        localStorage.setItem('vsphere_password', cleanedConfig.password);
        localStorage.setItem('vsphere_insecure', cleanedConfig.insecure.toString());
      } else {
        localStorage.removeItem('vsphere_vcenter');
        localStorage.removeItem('vsphere_datacenter');
        localStorage.removeItem('vsphere_username');
        localStorage.removeItem('vsphere_password');
        localStorage.removeItem('vsphere_insecure');
      }

      setCurrentStep('discover');
    } catch (err) {
      console.error('vSphere login error:', err);
      setError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setLoading(false);
    }
  };

  // Step 2: Select VM
  const handleVMSelect = (vm: VM) => {
    setSelectedVM(vm);
    setExportOptions(prev => ({
      ...prev,
      jobName: vm.name || '',
    }));
    setCurrentStep('export');
  };

  // Step 3: Submit Export Job
  const handleExportSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const jobData = {
        provider: 'vsphere',
        vm_identifier: selectedVM?.id,
        vm_name: selectedVM?.name,
        name: exportOptions.jobName,
        output_dir: exportOptions.outputDir,
        format: exportOptions.format,
        compress: exportOptions.compress,
        enable_pipeline: exportOptions.enablePipeline,
        // vSphere connection details
        vsphere_config: vSphereConfig,
      };

      const response = await fetch('/jobs/submit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(jobData),
      });

      if (!response.ok) {
        throw new Error('Failed to submit export job');
      }

      const result = await response.json();
      alert(`Job submitted successfully! Job ID: ${result.job_id || 'N/A'}`);

      // Reset workflow
      setCurrentStep('login');
      setSelectedVM(null);
      setVms([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed');
    } finally {
      setLoading(false);
    }
  };

  const handleBackToVMs = () => {
    setCurrentStep('discover');
    setSelectedVM(null);
  };

  const handleBackToLogin = () => {
    setCurrentStep('login');
    setVms([]);
    setSelectedVM(null);
  };

  const getStatusColor = (status: string): string => {
    const s = status.toLowerCase();
    if (s.includes('on') || s.includes('running')) return '#4caf50';
    if (s.includes('off') || s.includes('stopped')) return '#f44336';
    return '#ff9800';
  };

  // Render Step 1: vSphere Login
  const renderLoginStep = () => (
    <div style={styles.stepContainer}>
      <h2 style={styles.stepTitle}>Step 1: Connect to vSphere</h2>
      <p style={styles.stepDescription}>
        Enter your vCenter credentials to discover virtual machines
      </p>

      <form onSubmit={handleVSphereLogin} style={styles.form}>
        <div style={styles.formGroup}>
          <label style={styles.label}>
            vCenter Server <span style={styles.required}>*</span>
          </label>
          <input
            type="text"
            value={vSphereConfig.vcenter}
            onChange={(e) => setVSphereConfig({ ...vSphereConfig, vcenter: e.target.value })}
            onBlur={(e) => setVSphereConfig({ ...vSphereConfig, vcenter: e.target.value.trim().replace(/\s+/g, '') })}
            placeholder="vcenter.example.com"
            required
            style={styles.input}
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            Datacenter <span style={{ ...styles.optional, fontSize: '11px', fontWeight: '400', color: '#6b7280' }}>(optional - leave empty to discover all)</span>
          </label>
          <input
            type="text"
            value={vSphereConfig.datacenter}
            onChange={(e) => setVSphereConfig({ ...vSphereConfig, datacenter: e.target.value })}
            onBlur={(e) => setVSphereConfig({ ...vSphereConfig, datacenter: e.target.value.trim() })}
            placeholder="Leave empty to auto-discover all datacenters"
            style={styles.input}
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            Username <span style={styles.required}>*</span>
          </label>
          <input
            type="text"
            value={vSphereConfig.username}
            onChange={(e) => setVSphereConfig({ ...vSphereConfig, username: e.target.value })}
            onBlur={(e) => setVSphereConfig({ ...vSphereConfig, username: e.target.value.trim() })}
            placeholder="administrator@vsphere.local"
            required
            style={styles.input}
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            Password <span style={styles.required}>*</span>
          </label>
          <input
            type={showPassword ? 'text' : 'password'}
            value={vSphereConfig.password}
            onChange={(e) => setVSphereConfig({ ...vSphereConfig, password: e.target.value })}
            onBlur={(e) => setVSphereConfig({ ...vSphereConfig, password: e.target.value.trim() })}
            placeholder="Enter password"
            required
            style={styles.input}
          />
          <label style={{ ...styles.checkboxLabel, marginTop: '8px' }}>
            <input
              type="checkbox"
              checked={showPassword}
              onChange={(e) => setShowPassword(e.target.checked)}
              style={styles.checkbox}
            />
            Show password
          </label>
        </div>

        <div style={styles.formGroup}>
          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={vSphereConfig.insecure}
              onChange={(e) => setVSphereConfig({ ...vSphereConfig, insecure: e.target.checked })}
              style={styles.checkbox}
            />
            Skip SSL verification (for self-signed certificates)
          </label>
        </div>

        <div style={styles.formGroup}>
          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={rememberVSphere}
              onChange={(e) => setRememberVSphere(e.target.checked)}
              style={styles.checkbox}
            />
            Remember vSphere credentials
          </label>
        </div>

        {error && (
          <div style={styles.errorBox}>
            <strong>⚠️ Error:</strong> {error}
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          style={{
            ...styles.submitButton,
            ...(loading ? styles.submitButtonDisabled : {}),
          }}
        >
          {loading ? '🔄 Connecting...' : '🔍 Connect & Discover VMs'}
        </button>
      </form>
    </div>
  );

  // Render Step 2: VM Discovery
  const renderDiscoverStep = () => (
    <div style={styles.stepContainer}>
      <div style={styles.stepHeader}>
        <div>
          <h2 style={styles.stepTitle}>Step 2: Select Virtual Machine</h2>
          <p style={styles.stepDescription}>
            {vms.length} VM{vms.length !== 1 ? 's' : ''} discovered from {vSphereConfig.vcenter}
          </p>
        </div>
        <button onClick={handleBackToLogin} style={styles.backButton}>
          ← Back to Login
        </button>
      </div>

      {vms.length === 0 ? (
        <div style={styles.emptyState}>
          <p style={styles.emptyIcon}>📭</p>
          <p style={styles.emptyText}>No virtual machines found</p>
        </div>
      ) : (
        <div style={styles.vmGrid}>
          {vms.map((vm) => (
            <div
              key={vm.id}
              style={styles.vmCard}
              onClick={() => handleVMSelect(vm)}
            >
              <div style={styles.vmCardHeader}>
                <h3 style={styles.vmCardTitle}>{vm.name}</h3>
                <span
                  style={{
                    ...styles.statusBadge,
                    backgroundColor: getStatusColor(vm.power_state),
                  }}
                >
                  {vm.power_state}
                </span>
              </div>

              <div style={styles.vmCardBody}>
                <div style={styles.vmCardRow}>
                  <span style={styles.vmCardLabel}>ID:</span>
                  <span style={styles.vmCardValue}>{vm.id}</span>
                </div>
                {vm.cpu_count && (
                  <div style={styles.vmCardRow}>
                    <span style={styles.vmCardLabel}>CPU:</span>
                    <span style={styles.vmCardValue}>{vm.cpu_count} vCPU</span>
                  </div>
                )}
                {vm.memory_mb && (
                  <div style={styles.vmCardRow}>
                    <span style={styles.vmCardLabel}>Memory:</span>
                    <span style={styles.vmCardValue}>
                      {(vm.memory_mb / 1024).toFixed(1)} GB
                    </span>
                  </div>
                )}
                {vm.os && (
                  <div style={styles.vmCardRow}>
                    <span style={styles.vmCardLabel}>OS:</span>
                    <span style={styles.vmCardValue}>{vm.os}</span>
                  </div>
                )}
                {vm.ip_address && (
                  <div style={styles.vmCardRow}>
                    <span style={styles.vmCardLabel}>IP:</span>
                    <span style={styles.vmCardValue}>{vm.ip_address}</span>
                  </div>
                )}
              </div>

              <button
                style={styles.selectButton}
                onClick={(e) => {
                  e.stopPropagation();
                  handleVMSelect(vm);
                }}
              >
                Select for Export →
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  // Render Step 3: Export Options
  const renderExportStep = () => (
    <div style={styles.stepContainer}>
      <div style={styles.stepHeader}>
        <div>
          <h2 style={styles.stepTitle}>Step 3: Configure Export Options</h2>
          <p style={styles.stepDescription}>
            Export {selectedVM?.name} from vSphere to local storage
          </p>
        </div>
        <button onClick={handleBackToVMs} style={styles.backButton}>
          ← Back to VMs
        </button>
      </div>

      {/* Selected VM Summary */}
      <div style={styles.selectedVMBox}>
        <h3 style={styles.selectedVMTitle}>✓ Selected Virtual Machine</h3>
        <div style={styles.selectedVMGrid}>
          <div><strong>Name:</strong> {selectedVM?.name}</div>
          <div><strong>ID:</strong> {selectedVM?.id}</div>
          <div><strong>Status:</strong> {selectedVM?.power_state}</div>
          {selectedVM?.datacenter && <div><strong>Datacenter:</strong> {selectedVM.datacenter}</div>}
        </div>
      </div>

      {/* Export Options Form */}
      <form onSubmit={handleExportSubmit} style={styles.form}>
        <h3 style={styles.sectionTitle}>Export Options</h3>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            Job Name <span style={styles.required}>*</span>
          </label>
          <input
            type="text"
            value={exportOptions.jobName}
            onChange={(e) => setExportOptions({ ...exportOptions, jobName: e.target.value })}
            placeholder="my-export-job"
            required
            style={styles.input}
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            Output Directory <span style={styles.required}>*</span>
          </label>
          <input
            type="text"
            value={exportOptions.outputDir}
            onChange={(e) => setExportOptions({ ...exportOptions, outputDir: e.target.value })}
            placeholder="/tmp/exports"
            required
            style={styles.input}
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>
            Format <span style={styles.required}>*</span>
          </label>
          <select
            value={exportOptions.format}
            onChange={(e) => setExportOptions({ ...exportOptions, format: e.target.value })}
            style={styles.input}
          >
            <option value="ova">OVA (Open Virtual Appliance)</option>
            <option value="ovf">OVF (Open Virtualization Format)</option>
            <option value="vmdk">VMDK (Virtual Machine Disk)</option>
          </select>
        </div>

        <div style={styles.formGroup}>
          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={exportOptions.compress}
              onChange={(e) => setExportOptions({ ...exportOptions, compress: e.target.checked })}
              style={styles.checkbox}
            />
            Compress output
          </label>
        </div>

        <h3 style={styles.sectionTitle}>Pipeline Integration (hyper2kvm + libvirt)</h3>

        <div style={styles.formGroup}>
          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={exportOptions.enablePipeline}
              onChange={(e) => setExportOptions({ ...exportOptions, enablePipeline: e.target.checked })}
              style={styles.checkbox}
            />
            Enable hyper2kvm pipeline after export
          </label>
          <p style={styles.helpText}>
            Automatically convert exported VM to KVM format and import to libvirt
          </p>
        </div>

        {error && (
          <div style={styles.errorBox}>
            <strong>⚠️ Error:</strong> {error}
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          style={{
            ...styles.submitButton,
            ...(loading ? styles.submitButtonDisabled : {}),
          }}
        >
          {loading ? '⏳ Submitting...' : '📤 Submit Export Job'}
        </button>
      </form>
    </div>
  );

  return (
    <div style={styles.container}>
      {/* Progress Indicator */}
      <div style={styles.progressBar}>
        <div style={{
          ...styles.progressStep,
          ...(currentStep === 'login' ? styles.progressStepActive : styles.progressStepComplete)
        }}>
          {currentStep === 'login' ? '1' : '✓'} vSphere Login
        </div>
        <div style={styles.progressLine} />
        <div style={{
          ...styles.progressStep,
          ...(currentStep === 'discover' ? styles.progressStepActive : currentStep === 'export' ? styles.progressStepComplete : {})
        }}>
          {currentStep === 'export' ? '✓' : '2'} Discover VMs
        </div>
        <div style={styles.progressLine} />
        <div style={{
          ...styles.progressStep,
          ...(currentStep === 'export' ? styles.progressStepActive : {})
        }}>
          3 Export Options
        </div>
      </div>

      {/* Step Content */}
      {currentStep === 'login' && renderLoginStep()}
      {currentStep === 'discover' && renderDiscoverStep()}
      {currentStep === 'export' && renderExportStep()}
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    padding: '16px',
    maxWidth: '800px',
    margin: '0 auto',
  },
  progressBar: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: '16px',
    padding: '12px',
    backgroundColor: '#f8f9fa',
    borderRadius: '6px',
  },
  progressStep: {
    padding: '6px 14px',
    backgroundColor: '#e9ecef',
    color: '#6c757d',
    borderRadius: '16px',
    fontWeight: '500',
    fontSize: '12px',
  },
  progressStepActive: {
    backgroundColor: '#f0583a',
    color: '#fff',
  },
  progressStepComplete: {
    backgroundColor: '#4caf50',
    color: '#fff',
  },
  progressLine: {
    width: '40px',
    height: '2px',
    backgroundColor: '#dee2e6',
    margin: '0 8px',
  },
  stepContainer: {
    backgroundColor: '#fff',
    padding: '20px',
    borderRadius: '6px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
  },
  stepHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: '16px',
  },
  stepTitle: {
    margin: '0 0 6px 0',
    fontSize: '18px',
    color: '#212529',
    fontWeight: '600',
  },
  stepDescription: {
    margin: 0,
    fontSize: '13px',
    color: '#6c757d',
  },
  backButton: {
    padding: '6px 12px',
    backgroundColor: '#6c757d',
    color: '#fff',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '12px',
    fontWeight: '500',
  },
  form: {
    marginTop: '16px',
  },
  formGroup: {
    marginBottom: '14px',
  },
  label: {
    display: 'block',
    marginBottom: '6px',
    fontWeight: '500',
    color: '#495057',
    fontSize: '13px',
  },
  required: {
    color: '#f44336',
  },
  input: {
    width: '100%',
    maxWidth: '100%',
    padding: '8px 12px',
    border: '1px solid #ced4da',
    borderRadius: '4px',
    fontSize: '13px',
    boxSizing: 'border-box',
    transition: 'border-color 0.15s',
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '12px',
    color: '#495057',
  },
  checkbox: {
    width: '16px',
    height: '16px',
    cursor: 'pointer',
  },
  helpText: {
    margin: '6px 0 0 24px',
    fontSize: '11px',
    color: '#6c757d',
    fontStyle: 'italic',
  },
  sectionTitle: {
    margin: '20px 0 12px 0',
    fontSize: '15px',
    color: '#212529',
    fontWeight: '600',
    borderBottom: '1px solid #e9ecef',
    paddingBottom: '8px',
  },
  errorBox: {
    padding: '10px 12px',
    backgroundColor: '#f8d7da',
    border: '1px solid #f5c2c7',
    borderRadius: '4px',
    color: '#842029',
    marginBottom: '12px',
    fontSize: '13px',
  },
  submitButton: {
    width: '100%',
    maxWidth: '200px',
    padding: '10px 16px',
    backgroundColor: '#f0583a',
    color: '#fff',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: '500',
    marginTop: '16px',
  },
  submitButtonDisabled: {
    backgroundColor: '#adb5bd',
    cursor: 'not-allowed',
  },
  vmGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
    gap: '12px',
    marginTop: '12px',
  },
  vmCard: {
    padding: '14px',
    border: '1px solid #dee2e6',
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'all 0.2s',
    backgroundColor: '#fff',
  },
  vmCardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '10px',
  },
  vmCardTitle: {
    margin: 0,
    fontSize: '14px',
    color: '#212529',
    fontWeight: '600',
  },
  statusBadge: {
    padding: '2px 8px',
    borderRadius: '10px',
    color: '#fff',
    fontSize: '11px',
    fontWeight: '500',
  },
  vmCardBody: {
    marginBottom: '10px',
  },
  vmCardRow: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: '4px 0',
    borderBottom: '1px solid #f8f9fa',
  },
  vmCardLabel: {
    fontWeight: '500',
    color: '#6c757d',
    fontSize: '12px',
  },
  vmCardValue: {
    color: '#495057',
    fontSize: '12px',
  },
  selectButton: {
    width: '100%',
    padding: '8px',
    backgroundColor: '#f0583a',
    color: '#fff',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '12px',
    fontWeight: '500',
  },
  selectedVMBox: {
    padding: '12px 14px',
    backgroundColor: '#d1e7dd',
    border: '1px solid #a3cfbb',
    borderRadius: '6px',
    marginBottom: '16px',
  },
  selectedVMTitle: {
    margin: '0 0 10px 0',
    fontSize: '14px',
    color: '#0f5132',
    fontWeight: '600',
  },
  selectedVMGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
    gap: '8px',
    fontSize: '12px',
  },
  emptyState: {
    textAlign: 'center',
    padding: '40px 20px',
    color: '#6c757d',
  },
  emptyIcon: {
    fontSize: '48px',
    margin: '0 0 8px 0',
  },
  emptyText: {
    fontSize: '14px',
    margin: 0,
  },
};

export default VSphereExportWorkflow;
