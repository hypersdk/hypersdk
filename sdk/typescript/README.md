# HyperSDK TypeScript Client

TypeScript/JavaScript client library for the HyperSDK VM migration and export platform.

## Features

- 🚀 Modern async/await API
- 📘 Full TypeScript type definitions
- 🔐 Built-in authentication support
- ✅ Comprehensive error handling
- 🌐 Works in Node.js and browsers
- 📦 Support for all HyperSDK operations:
  - Job submission and monitoring
  - VM discovery and operations
  - Schedule management
  - Webhook configuration
  - Libvirt integration
  - Hyper2KVM conversion

## Installation

```bash
npm install @hypersdk/client
```

Or with yarn:

```bash
yarn add @hypersdk/client
```

## Quick Start

### Basic Usage (TypeScript)

```typescript
import { HyperSDK, JobDefinition, JobStatus, ExportFormat } from '@hypersdk/client';

// Initialize client
const client = new HyperSDK({
  baseURL: 'http://localhost:8080',
  timeout: 30000,
});

async function main() {
  try {
    // Login (if authentication is enabled)
    await client.login('admin', 'password');

    // Check daemon status
    const status = await client.status();
    console.log(`Daemon version: ${status.version}`);
    console.log(`Running jobs: ${status.running_jobs}`);

    // Submit a VM export job
    const jobDef: JobDefinition = {
      vm_path: '/Datacenter/vm/my-virtual-machine',
      output_dir: '/exports',
      vcenter: {
        server: 'vcenter.example.com',
        username: 'administrator@vsphere.local',
        password: 'your-password',
        insecure: true,
      },
      format: ExportFormat.OVF,
      compress: true,
    };

    const jobId = await client.submitJob(jobDef);
    console.log(`Job submitted: ${jobId}`);

    // Monitor job progress
    const job = await client.getJob(jobId);
    console.log(`Job status: ${job.status}`);

    if (job.progress) {
      console.log(`Progress: ${job.progress.percent_complete}%`);
      console.log(`Phase: ${job.progress.phase}`);
    }
  } catch (error) {
    console.error('Error:', error);
  }
}

main();
```

### Basic Usage (JavaScript)

```javascript
const { HyperSDK } = require('@hypersdk/client');

const client = new HyperSDK('http://localhost:8080');

client.submitJob({
  vm_path: '/Datacenter/vm/my-vm',
  output_dir: '/exports',
  format: 'ovf',
}).then((jobId) => {
  console.log('Job submitted:', jobId);
  return client.getJob(jobId);
}).then((job) => {
  console.log('Job status:', job.status);
}).catch((error) => {
  console.error('Error:', error);
});
```

### List All Jobs

```typescript
// Get all jobs
const jobs = await client.listJobs(true);

for (const job of jobs) {
  console.log(`Job ${job.definition.id}: ${job.status}`);
  if (job.progress) {
    console.log(`  Progress: ${job.progress.percent_complete}%`);
  }
}
```

### Filter Jobs by Status

```typescript
import { JobStatus } from '@hypersdk/client';

// Get only running jobs
const runningJobs = await client.queryJobs({
  status: [JobStatus.RUNNING],
});

for (const job of runningJobs) {
  const progress = await client.getJobProgress(job.definition.id!);
  console.log(`${job.definition.name}: ${progress.percent_complete}% complete`);
  console.log(`  ETA: ${progress.estimated_remaining}`);
}
```

### Cancel a Job

```typescript
const success = await client.cancelJob(jobId);
if (success) {
  console.log('Job cancelled successfully');
}
```

### Scheduled Jobs

```typescript
import { ScheduledJob } from '@hypersdk/client';

// Create a scheduled job (runs daily at 2 AM)
const schedule: ScheduledJob = {
  name: 'Daily VM Backup',
  description: 'Backup production VMs every night',
  schedule: '0 2 * * *', // Cron format
  job_template: {
    vm_path: '/Datacenter/vm/production-vm',
    output_dir: '/backups',
    format: ExportFormat.OVA,
    compress: true,
  },
  enabled: true,
  tags: ['backup', 'production'],
};

const createdSchedule = await client.createSchedule(schedule);
console.log(`Schedule created: ${createdSchedule.id}`);
console.log(`Next run: ${createdSchedule.next_run}`);

// List all schedules
const schedules = await client.listSchedules();
for (const sched of schedules) {
  console.log(`${sched.name}: ${sched.schedule} (enabled=${sched.enabled})`);
}

// Manually trigger a schedule
await client.triggerSchedule(createdSchedule.id!);
```

### Webhooks

```typescript
import { Webhook } from '@hypersdk/client';

// Add a webhook for job completion notifications
const webhook: Webhook = {
  url: 'https://myapp.example.com/webhook',
  events: ['job_completed', 'job_failed'],
  headers: {
    'Authorization': 'Bearer my-webhook-token',
    'X-Custom-Header': 'value',
  },
};

await client.addWebhook(webhook);

// Test the webhook
await client.testWebhook('https://myapp.example.com/webhook');
```

### VM Operations

```typescript
const vcenterConfig = {
  server: 'vcenter.example.com',
  username: 'admin',
  password: 'password',
  insecure: true,
};

// List VMs
const vms = await client.listVMs(vcenterConfig);
for (const vm of vms) {
  console.log(`VM: ${vm.name} - ${vm.power_state}`);
}

// Get VM details
const vmInfo = await client.getVMInfo(
  vcenterConfig,
  '/Datacenter/vm/my-vm'
);
console.log(`CPU: ${vmInfo.cpu}, Memory: ${vmInfo.memory_mb} MB`);

// Shutdown a VM
await client.shutdownVM(vcenterConfig, '/Datacenter/vm/my-vm');
```

### Libvirt Integration

```typescript
// List libvirt domains
const domains = await client.listDomains();
for (const domain of domains) {
  console.log(`Domain: ${domain.name} - ${domain.state}`);
}

// Start a domain
await client.startDomain('my-vm');

// Create a snapshot
await client.createSnapshot(
  'my-vm',
  'before-update',
  'Snapshot before system update'
);

// List snapshots
const snapshots = await client.listSnapshots('my-vm');
for (const snapshot of snapshots) {
  console.log(`Snapshot: ${snapshot.name}`);
}
```

### Hyper2KVM Conversion

```typescript
// Convert a VM
const conversionId = await client.convertVM(
  '/exports/vm.ovf',
  '/converted/vm.qcow2'
);

// Check conversion status
const conversionStatus = await client.getConversionStatus(conversionId);
console.log('Conversion status:', conversionStatus);
```

## Advanced Usage

### Custom Configuration

```typescript
const client = new HyperSDK({
  baseURL: 'https://hypersdk.example.com',
  apiKey: 'your-api-key',
  timeout: 60000, // 60 second timeout
  headers: {
    'X-Custom-Header': 'value',
  },
});
```

### Export with Advanced Options

```typescript
const jobDef: JobDefinition = {
  vm_path: '/Datacenter/vm/my-vm',
  output_dir: '/exports',
  options: {
    parallel_downloads: 8,
    remove_cdrom: true,
    show_individual_progress: true,
    enable_pipeline: true,
    pipeline_convert: true,
    pipeline_validate: true,
    libvirt_integration: true,
    libvirt_uri: 'qemu:///system',
    libvirt_pool: 'default',
  },
};

const jobId = await client.submitJob(jobDef);
```

### Batch Job Submission

```typescript
const jobs: JobDefinition[] = [
  { vm_path: '/Datacenter/vm/vm1', output_dir: '/exports' },
  { vm_path: '/Datacenter/vm/vm2', output_dir: '/exports' },
  { vm_path: '/Datacenter/vm/vm3', output_dir: '/exports' },
];

const jobIds = await client.submitJobs(jobs);
console.log(`Submitted ${jobIds.length} jobs`);
```

### Error Handling

```typescript
import {
  HyperSDKError,
  AuthenticationError,
  JobNotFoundError,
  APIError,
} from '@hypersdk/client';

try {
  const client = new HyperSDK('http://localhost:8080');
  await client.login('admin', 'wrong-password');
} catch (error) {
  if (error instanceof AuthenticationError) {
    console.error('Login failed:', error.message);
  } else if (error instanceof HyperSDKError) {
    console.error('SDK error:', error.message);
  } else {
    console.error('Unexpected error:', error);
  }
}

try {
  const job = await client.getJob('non-existent-job');
} catch (error) {
  if (error instanceof JobNotFoundError) {
    console.error('Job not found:', error.message);
  }
}

try {
  const jobId = await client.submitJob({ vm_path: '/invalid/path' });
} catch (error) {
  if (error instanceof APIError) {
    console.error('API error:', error.message);
    console.error('Status code:', error.statusCode);
    console.error('Response:', error.response);
  }
}
```

### Async Iteration Pattern

```typescript
async function monitorJob(jobId: string): Promise<void> {
  while (true) {
    const job = await client.getJob(jobId);

    console.log(`Status: ${job.status}`);

    if (job.progress) {
      console.log(`Progress: ${job.progress.percent_complete}%`);
    }

    if (
      job.status === JobStatus.COMPLETED ||
      job.status === JobStatus.FAILED ||
      job.status === JobStatus.CANCELLED
    ) {
      break;
    }

    await new Promise((resolve) => setTimeout(resolve, 2000));
  }
}
```

## API Reference

### Client Methods

#### Authentication
- `login(username, password)` - Login and obtain session token
- `logout()` - Logout and invalidate session

#### Health & Status
- `health()` - Check API health
- `status()` - Get daemon status
- `capabilities()` - Get export capabilities

#### Job Management
- `submitJob(jobDef)` - Submit a single job
- `submitJobs(jobDefs)` - Submit multiple jobs
- `getJob(jobId)` - Get job details
- `listJobs(all?)` - List all jobs
- `queryJobs(query)` - Query jobs with filters
- `cancelJob(jobId)` - Cancel a job
- `cancelJobs(jobIds)` - Cancel multiple jobs
- `getJobProgress(jobId)` - Get job progress
- `getJobLogs(jobId)` - Get job logs
- `getJobETA(jobId)` - Get job ETA

#### VM Operations
- `listVMs(vcenterConfig)` - List VMs
- `getVMInfo(vcenterConfig, vmPath)` - Get VM info
- `shutdownVM(vcenterConfig, vmPath)` - Shutdown VM

#### Schedule Management
- `listSchedules()` - List schedules
- `createSchedule(schedule)` - Create schedule
- `getSchedule(scheduleId)` - Get schedule
- `updateSchedule(scheduleId, schedule)` - Update schedule
- `deleteSchedule(scheduleId)` - Delete schedule
- `enableSchedule(scheduleId)` - Enable schedule
- `disableSchedule(scheduleId)` - Disable schedule
- `triggerSchedule(scheduleId)` - Trigger schedule

#### Webhook Management
- `listWebhooks()` - List webhooks
- `addWebhook(webhook)` - Add webhook
- `testWebhook(url)` - Test webhook
- `deleteWebhook(webhookId)` - Delete webhook

#### Libvirt Operations
- `listDomains()` - List domains
- `getDomain(name)` - Get domain
- `startDomain(name)` - Start domain
- `shutdownDomain(name)` - Shutdown domain
- `listSnapshots(domain)` - List snapshots
- `createSnapshot(domain, name, description?)` - Create snapshot

#### Hyper2KVM Integration
- `convertVM(sourcePath, outputPath)` - Convert VM
- `getConversionStatus(conversionId)` - Get conversion status

## Development

### Building

```bash
npm install
npm run build
```

### Running Tests

```bash
npm test
```

### Code Formatting

```bash
npm run format
```

### Linting

```bash
npm run lint
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

LGPL-3.0-or-later

## Support

- **Issues**: https://github.com/ssahani/hypersdk/issues
- **Documentation**: https://github.com/ssahani/hypersdk
