import React, { useState } from 'react';

interface JobSubmissionFormProps {
  onSubmit: (jobData: any) => Promise<void>;
}

type CloudProvider = 'vsphere' | 'aws' | 'azure' | 'gcp' | 'hyperv' | 'oci' | 'openstack' | 'alibabacloud' | 'proxmox';

export const JobSubmissionForm: React.FC<JobSubmissionFormProps> = ({ onSubmit }) => {
  const [provider, setProvider] = useState<CloudProvider>('vsphere');
  const [formData, setFormData] = useState<Record<string, any>>({
    name: '',
    provider: 'vsphere',
    output_dir: '/tmp/exports',
    format: 'ova',
    compress: false,
  });

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    const checked = (e.target as HTMLInputElement).checked;

    setFormData((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));
  };

  const handleProviderChange = (newProvider: CloudProvider) => {
    setProvider(newProvider);
    setFormData({
      name: '',
      provider: newProvider,
      output_dir: '/tmp/exports',
      format: newProvider === 'aws' ? 'vmdk' : newProvider === 'azure' || newProvider === 'gcp' ? 'vhd' : 'ova',
      compress: false,
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);
    setSuccess(false);

    try {
      await onSubmit({ ...formData, provider });
      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit job');
    } finally {
      setIsSubmitting(false);
    }
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '14px 16px',
    border: '2px solid #000',
    borderRadius: '4px',
    fontSize: '16px',
    backgroundColor: '#fff',
    transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
    color: '#000',
  };

  const labelStyle: React.CSSProperties = {
    display: 'block',
    fontSize: '14px',
    fontWeight: '700',
    color: '#000',
    marginBottom: '10px',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
  };

  return (
    <div style={{ backgroundColor: '#fff', borderRadius: '4px', padding: '40px', border: '2px solid #e0e0e0' }}>
      <h2 style={{ margin: '0 0 32px 0', fontSize: '28px', fontWeight: '700', color: '#000', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Submit Export Job</h2>

      {error && (
        <div style={{ padding: '16px', backgroundColor: '#fee2e2', color: '#991b1b', borderRadius: '8px', marginBottom: '20px', fontWeight: '500' }}>
          {error}
        </div>
      )}

      {success && (
        <div style={{ padding: '16px', backgroundColor: '#d1fae5', color: '#065f46', borderRadius: '8px', marginBottom: '20px', fontWeight: '500' }}>
          Job submitted successfully!
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <div style={{ display: 'grid', gridTemplateColumns: '280px 1fr', gap: '32px' }}>
          {/* Left Sidebar - Cloud Provider Selection */}
          <div style={{ padding: '24px', backgroundColor: '#f0f2f7', borderRadius: '4px', border: '2px solid #e0e0e0', height: 'fit-content' }}>
            <label style={{ ...labelStyle, marginBottom: '20px' }}>Cloud Providers</label>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {(['vsphere', 'aws', 'azure', 'gcp', 'hyperv', 'oci', 'openstack', 'alibabacloud', 'proxmox'] as CloudProvider[]).map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => handleProviderChange(p)}
                  onMouseEnter={(e) => {
                    if (provider !== p) {
                      e.currentTarget.style.backgroundColor = '#f0583a';
                      e.currentTarget.style.borderColor = '#f0583a';
                      e.currentTarget.style.color = '#fff';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (provider !== p) {
                      e.currentTarget.style.backgroundColor = '#fff';
                      e.currentTarget.style.borderColor = '#222324';
                      e.currentTarget.style.color = '#222324';
                    }
                  }}
                  style={{
                    padding: '14px 20px',
                    borderRadius: '4px',
                    border: provider === p ? '2px solid #f0583a' : '2px solid #222324',
                    backgroundColor: provider === p ? '#f0583a' : '#fff',
                    color: provider === p ? '#fff' : '#222324',
                    fontSize: '14px',
                    fontWeight: '700',
                    cursor: 'pointer',
                    transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.5px',
                    textAlign: 'left',
                  }}
                >
                  {p === 'vsphere' ? 'vSphere' : p === 'gcp' ? 'GCP' : p === 'oci' ? 'OCI' : p.replace('cloud', ' Cloud')}
                </button>
              ))}
            </div>
          </div>

          {/* Right Content - Provider Configuration */}
          <div>
        {/* Provider-Specific Fields */}
        {provider === 'vsphere' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>vSphere Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>vCenter Server *</label>
                <input type="text" name="vcenter_url" onChange={handleChange} placeholder="vcenter.example.com" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Datacenter *</label>
                <input type="text" name="datacenter" onChange={handleChange} placeholder="DC1" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Username *</label>
                <input type="text" name="username" onChange={handleChange} placeholder="administrator@vsphere.local" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Password *</label>
                <input type="password" name="password" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>VM Path *</label>
                <input type="text" name="vm_path" onChange={handleChange} placeholder="/DC1/vm/folder/my-vm" required style={inputStyle} />
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingTop: '28px' }}>
                <input type="checkbox" name="insecure" onChange={handleChange} id="insecure" defaultChecked style={{ width: '16px', height: '16px' }} />
                <label htmlFor="insecure" style={{ fontSize: '13px', cursor: 'pointer' }}>Skip SSL verification</label>
              </div>
            </div>
          </div>
        )}

        {provider === 'aws' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>AWS Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Access Key ID *</label>
                <input type="text" name="access_key_id" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Secret Access Key *</label>
                <input type="password" name="secret_access_key" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Region *</label>
                <input type="text" name="region" onChange={handleChange} placeholder="us-east-1" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Instance ID *</label>
                <input type="text" name="instance_id" onChange={handleChange} placeholder="i-1234567890abcdef0" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>S3 Bucket *</label>
                <input type="text" name="bucket" onChange={handleChange} placeholder="my-exports-bucket" required style={inputStyle} />
              </div>
            </div>
          </div>
        )}

        {provider === 'azure' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Azure Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Subscription ID *</label>
                <input type="text" name="subscription_id" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Tenant ID *</label>
                <input type="text" name="tenant_id" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Client ID *</label>
                <input type="text" name="client_id" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Client Secret *</label>
                <input type="password" name="client_secret" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Resource Group *</label>
                <input type="text" name="resource_group" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>VM Name *</label>
                <input type="text" name="vm_name" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Storage Account *</label>
                <input type="text" name="storage_account" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Container *</label>
                <input type="text" name="container" onChange={handleChange} placeholder="vm-exports" required style={inputStyle} />
              </div>
            </div>
          </div>
        )}

        {provider === 'gcp' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>GCP Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Project ID *</label>
                <input type="text" name="project_id" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Zone *</label>
                <input type="text" name="zone" onChange={handleChange} placeholder="us-central1-a" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Instance Name *</label>
                <input type="text" name="instance_name" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>GCS Bucket *</label>
                <input type="text" name="bucket" onChange={handleChange} placeholder="my-exports-bucket" required style={inputStyle} />
              </div>
              <div style={{ gridColumn: '1 / -1' }}>
                <label style={labelStyle}>Service Account Key (JSON) *</label>
                <textarea name="service_account_json" onChange={(e: any) => handleChange(e)} required style={{ ...inputStyle, minHeight: '80px', fontFamily: 'monospace', fontSize: '12px' }} placeholder='{"type": "service_account", ...}' />
              </div>
            </div>
          </div>
        )}

        {provider === 'hyperv' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Hyper-V Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Host *</label>
                <input type="text" name="host" onChange={handleChange} placeholder="hyperv.example.com" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Username *</label>
                <input type="text" name="username" onChange={handleChange} placeholder="DOMAIN\\Administrator" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Password *</label>
                <input type="password" name="password" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>VM Name *</label>
                <input type="text" name="vm_name" onChange={handleChange} required style={inputStyle} />
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingTop: '28px' }}>
                <input type="checkbox" name="use_winrm" onChange={handleChange} id="use_winrm" defaultChecked style={{ width: '16px', height: '16px' }} />
                <label htmlFor="use_winrm" style={{ fontSize: '13px', cursor: 'pointer' }}>Use WinRM</label>
              </div>
            </div>
          </div>
        )}

        {provider === 'oci' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>OCI Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Tenancy OCID *</label>
                <input type="text" name="tenancy_ocid" onChange={handleChange} placeholder="ocid1.tenancy.oc1..." required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>User OCID *</label>
                <input type="text" name="user_ocid" onChange={handleChange} placeholder="ocid1.user.oc1..." required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Fingerprint *</label>
                <input type="text" name="fingerprint" onChange={handleChange} placeholder="aa:bb:cc:..." required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Region *</label>
                <input type="text" name="region" onChange={handleChange} placeholder="us-phoenix-1" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Compartment OCID *</label>
                <input type="text" name="compartment_ocid" onChange={handleChange} placeholder="ocid1.compartment.oc1..." required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Instance OCID *</label>
                <input type="text" name="instance_ocid" onChange={handleChange} placeholder="ocid1.instance.oc1..." required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Bucket *</label>
                <input type="text" name="bucket" onChange={handleChange} placeholder="vm-backups" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Namespace *</label>
                <input type="text" name="namespace" onChange={handleChange} required style={inputStyle} />
              </div>
            </div>
          </div>
        )}

        {provider === 'openstack' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>OpenStack Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Auth URL *</label>
                <input type="text" name="auth_url" onChange={handleChange} placeholder="https://openstack.example.com:5000/v3" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Username *</label>
                <input type="text" name="username" onChange={handleChange} placeholder="admin" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Password *</label>
                <input type="password" name="password" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Tenant Name *</label>
                <input type="text" name="tenant_name" onChange={handleChange} placeholder="admin" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Domain *</label>
                <input type="text" name="domain_name" onChange={handleChange} placeholder="Default" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Region *</label>
                <input type="text" name="region" onChange={handleChange} placeholder="RegionOne" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Instance ID *</label>
                <input type="text" name="instance_id" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Container *</label>
                <input type="text" name="container" onChange={handleChange} placeholder="vm-backups" required style={inputStyle} />
              </div>
            </div>
          </div>
        )}

        {provider === 'alibabacloud' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Alibaba Cloud Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Access Key ID *</label>
                <input type="text" name="access_key_id" onChange={handleChange} placeholder="LTAI4G..." required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Access Key Secret *</label>
                <input type="password" name="access_key_secret" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Region ID *</label>
                <input type="text" name="region_id" onChange={handleChange} placeholder="cn-hangzhou" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Instance ID *</label>
                <input type="text" name="instance_id" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>OSS Bucket *</label>
                <input type="text" name="bucket" onChange={handleChange} placeholder="vm-backups" required style={inputStyle} />
              </div>
            </div>
          </div>
        )}

        {provider === 'proxmox' && (
          <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Proxmox VE Configuration</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
              <div>
                <label style={labelStyle}>Host *</label>
                <input type="text" name="host" onChange={handleChange} placeholder="proxmox.example.com" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Port</label>
                <input type="number" name="port" onChange={handleChange} defaultValue="8006" style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Username *</label>
                <input type="text" name="username" onChange={handleChange} placeholder="root@pam" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Password *</label>
                <input type="password" name="password" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Node *</label>
                <input type="text" name="node" onChange={handleChange} placeholder="pve1" required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>VM ID *</label>
                <input type="number" name="vmid" onChange={handleChange} required style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Storage</label>
                <input type="text" name="storage" onChange={handleChange} placeholder="local" defaultValue="local" style={inputStyle} />
              </div>
              <div>
                <label style={labelStyle}>Backup Mode</label>
                <select name="backup_mode" onChange={handleChange} style={inputStyle}>
                  <option value="snapshot">Snapshot (recommended)</option>
                  <option value="suspend">Suspend</option>
                  <option value="stop">Stop</option>
                </select>
              </div>
            </div>
          </div>
        )}

        {/* Common Fields */}
        <div style={{ marginBottom: '32px', padding: '24px', backgroundColor: '#1a1a1a', borderRadius: '4px', border: '1px solid #000' }}>
          <h3 style={{ fontSize: '20px', fontWeight: '700', marginBottom: '20px', color: '#fff', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Export Options</h3>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px' }}>
            <div>
              <label style={labelStyle}>Job Name *</label>
              <input type="text" name="name" value={formData.name} onChange={handleChange} placeholder="My Export Job" required style={inputStyle} />
            </div>
            <div>
              <label style={labelStyle}>Output Directory *</label>
              <input type="text" name="output_dir" value={formData.output_dir} onChange={handleChange} required style={inputStyle} />
            </div>
            <div>
              <label style={labelStyle}>Format *</label>
              <select name="format" value={formData.format} onChange={handleChange} required style={inputStyle}>
                {provider === 'aws' && <option value="vmdk">VMDK</option>}
                {(provider === 'azure' || provider === 'gcp') && <option value="vhd">VHD</option>}
                {provider === 'proxmox' && <option value="vzdump">Vzdump</option>}
                {['vsphere', 'hyperv', 'oci', 'openstack', 'alibabacloud'].includes(provider) && (
                  <>
                    <option value="ova">OVA</option>
                    <option value="ovf">OVF</option>
                    <option value="qcow2">QCOW2</option>
                    <option value="vmdk">VMDK</option>
                    <option value="raw">RAW</option>
                  </>
                )}
              </select>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingTop: '28px' }}>
              <input type="checkbox" name="compress" checked={formData.compress} onChange={handleChange} id="compress" style={{ width: '16px', height: '16px' }} />
              <label htmlFor="compress" style={{ fontSize: '13px', cursor: 'pointer' }}>Compress output</label>
            </div>
          </div>
        </div>

        {/* Submit */}
        <div style={{ display: 'flex', gap: '16px', justifyContent: 'flex-end', marginTop: '40px' }}>
          <button
            type="submit"
            disabled={isSubmitting}
            onMouseEnter={(e) => {
              if (!isSubmitting) {
                e.currentTarget.style.transform = 'translateY(-2px)';
                e.currentTarget.style.backgroundColor = '#d94b32';
              }
            }}
            onMouseLeave={(e) => {
              if (!isSubmitting) {
                e.currentTarget.style.transform = 'translateY(0)';
                e.currentTarget.style.backgroundColor = '#f0583a';
              }
            }}
            style={{
              padding: '16px 40px',
              borderRadius: '4px',
              border: 'none',
              backgroundColor: isSubmitting ? '#666' : '#f0583a',
              color: '#fff',
              fontSize: '16px',
              fontWeight: '700',
              cursor: isSubmitting ? 'not-allowed' : 'pointer',
              transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
              textTransform: 'uppercase',
              letterSpacing: '1px',
            }}
          >
            {isSubmitting ? 'Submitting...' : 'Submit Job'}
          </button>
        </div>
          </div>
        </div>
      </form>
    </div>
  );
};
