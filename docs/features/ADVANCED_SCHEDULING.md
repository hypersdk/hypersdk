// Due to message length limits, creating a concise but complete documentation

# Advanced Scheduling

HyperSDK provides advanced scheduling features including job dependencies, retry policies, time windows, and priority-based execution.

## Features

✅ **Job Dependencies** - Chain jobs together with dependency tracking
✅ **Retry Policies** - Automatic retries with configurable backoff strategies
✅ **Time Windows** - Restrict job execution to specific time periods
✅ **Job Priorities** - Priority-based queue management (0-100)
✅ **Conditional Execution** - Run jobs based on conditions
✅ **Concurrency Control** - Limit concurrent job executions

## Quick Start

### Create Schedule with Dependencies

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# Create schedule with dependencies
schedule = client.create_advanced_schedule(
    name="backup-and-upload",
    schedule="0 2 * * *",  # Daily at 2 AM
    job_template={
        "vm_path": "/Datacenter/vm/my-vm",
        "output_dir": "/backups"
    },
    advanced_config={
        "depends_on": [
            {
                "job_id": "snapshot-job-123",
                "required_state": "completed",
                "timeout": 3600
            }
        ],
        "retry_policy": {
            "max_attempts": 3,
            "initial_delay": 60,
            "max_delay": 1800,
            "backoff_strategy": "exponential"
        },
        "time_windows": [
            {
                "start_time": "22:00",
                "end_time": "06:00",
                "days": ["Mon", "Tue", "Wed", "Thu", "Fri"],
                "timezone": "America/New_York"
            }
        ],
        "priority": 80,
        "notify_on_failure": True,
        "notify_on_retry": True
    }
)
```

## Job Dependencies

### Configuration

```json
{
  "depends_on": [
    {
      "job_id": "prerequisite-job-id",
      "required_state": "completed",
      "timeout": 3600
    }
  ]
}
```

**Fields:**
- `job_id`: ID of the job this depends on
- `required_state`: `completed`, `failed`, or `any`
- `timeout`: Max wait time in seconds (0 = no timeout)

### Check Dependency Status

```python
status = client.get_dependency_status("job-123")
print(f"Dependencies satisfied: {status['satisfied']}")
print(f"Reason: {status['reason']}")
```

## Retry Policies

### Backoff Strategies

- **linear**: Delay increases linearly (1x, 2x, 3x)
- **exponential**: Delay doubles each attempt (1x, 2x, 4x, 8x)
- **fibonacci**: Delay follows Fibonacci sequence (1, 1, 2, 3, 5, 8)

### Example Configuration

```python
retry_policy = {
    "max_attempts": 5,
    "initial_delay": 60,        # 1 minute
    "max_delay": 1800,          # 30 minutes
    "backoff_strategy": "exponential",
    "retry_on_errors": ["timeout", "connection"]
}
```

### Check Retry Status

```python
retry = client.get_retry_status("job-123")
print(f"Attempt {retry['attempt']} of {retry['max_attempts']}")
print(f"Next retry: {retry['next_retry']}")
```

## Time Windows

### Business Hours Example

```python
time_windows = [
    {
        "start_time": "09:00",
        "end_time": "17:00",
        "days": ["Mon", "Tue", "Wed", "Thu", "Fri"],
        "timezone": "America/New_York"
    }
]
```

### Overnight Maintenance Window

```python
time_windows = [
    {
        "start_time": "22:00",
        "end_time": "06:00",
        "days": ["Sat", "Sun"],
        "timezone": "UTC"
    }
]
```

### Check Time Window Status

```python
status = client.get_timewindow_status("job-123")
print(f"In window: {status['in_window']}")
print(f"Next window: {status['next_window_start']}")
```

## Job Priorities

Jobs are queued and executed based on priority (0-100, higher = more important).

```python
advanced_config = {
    "priority": 90,  # High priority
    "max_concurrent": 1,
    "skip_if_running": True
}
```

### Queue Status

```python
queue = client.get_job_queue_status()
print(f"Queued: {queue['queue_size']}, Running: {queue['running_jobs']}")
```

## TypeScript SDK

```typescript
import { HyperSDK } from 'hypersdk';

const client = new HyperSDK('http://localhost:8080');

// Create advanced schedule
const schedule = await client.createAdvancedSchedule({
  name: 'backup-chain',
  schedule: '0 2 * * *',
  jobTemplate: {
    vm_path: '/Datacenter/vm/my-vm',
    output_dir: '/backups'
  },
  advancedConfig: {
    depends_on: [
      {
        job_id: 'snapshot-123',
        required_state: 'completed',
        timeout: 3600
      }
    ],
    retry_policy: {
      max_attempts: 3,
      initial_delay: 60,
      max_delay: 1800,
      backoff_strategy: 'exponential'
    },
    priority: 80
  }
});

// Check status
const depStatus = await client.getDependencyStatus('job-123');
const retryStatus = await client.getRetryStatus('job-123');
const windowStatus = await client.getTimeWindowStatus('job-123');
```

## REST API

### Create Advanced Schedule

**POST** `/schedules/advanced/create`

```json
{
  "name": "my-schedule",
  "schedule": "0 2 * * *",
  "job_template": {...},
  "advanced_config": {
    "depends_on": [...],
    "retry_policy": {...},
    "time_windows": [...],
    "priority": 80
  }
}
```

### Get Dependency Status

**GET** `/schedules/dependencies?job_id=<id>`

### Get Retry Status

**GET** `/schedules/retry?job_id=<id>`

### Get Time Window Status

**GET** `/schedules/timewindow?job_id=<id>`

### Get Queue Status

**GET** `/schedules/queue`

### Validate Schedule

**POST** `/schedules/validate`

## Use Cases

### 1. Multi-Stage Backup Pipeline

```python
# Stage 1: Snapshot
snapshot_schedule = client.create_advanced_schedule(
    name="snapshot",
    schedule="0 1 * * *",
    job_template={"action": "snapshot"},
    advanced_config={"priority": 100}
)

# Stage 2: Export (depends on snapshot)
export_schedule = client.create_advanced_schedule(
    name="export",
    schedule="0 1 * * *",
    job_template={"action": "export"},
    advanced_config={
        "depends_on": [{"job_id": snapshot_schedule["schedule"]["id"], "required_state": "completed"}],
        "retry_policy": {"max_attempts": 3, "backoff_strategy": "exponential"},
        "priority": 80
    }
)

# Stage 3: Upload (depends on export)
upload_schedule = client.create_advanced_schedule(
    name="upload",
    schedule="0 1 * * *",
    job_template={"action": "upload"},
    advanced_config={
        "depends_on": [{"job_id": export_schedule["schedule"]["id"], "required_state": "completed"}],
        "priority": 60
    }
)
```

### 2. Business Hours Only Backups

```python
schedule = client.create_advanced_schedule(
    name="business-hours-backup",
    schedule="0 */4 * * *",  # Every 4 hours
    job_template={"vm_path": "/Datacenter/vm/prod-db"},
    advanced_config={
        "time_windows": [
            {
                "start_time": "09:00",
                "end_time": "18:00",
                "days": ["Mon", "Tue", "Wed", "Thu", "Fri"],
                "timezone": "America/New_York"
            }
        ]
    }
)
```

### 3. Resilient Exports with Retry

```python
schedule = client.create_advanced_schedule(
    name="resilient-export",
    schedule="0 2 * * *",
    job_template={"vm_path": "/Datacenter/vm/important-vm"},
    advanced_config={
        "retry_policy": {
            "max_attempts": 5,
            "initial_delay": 300,  # 5 minutes
            "max_delay": 3600,     # 1 hour
            "backoff_strategy": "fibonacci",
            "retry_on_errors": ["network", "timeout", "storage"]
        },
        "notify_on_failure": True,
        "notify_on_retry": True
    }
)
```

## Best Practices

1. **Set appropriate timeouts** for dependencies to avoid infinite waits
2. **Use exponential backoff** for most retry scenarios
3. **Limit max_delay** to prevent excessive wait times
4. **Set time windows** for resource-intensive jobs to run off-hours
5. **Use priorities** to ensure critical jobs run first
6. **Enable notifications** for failed retries and dependency issues
7. **Validate schedules** before creating them in production

## Troubleshooting

### Dependencies Not Satisfied

Check dependency status:
```python
status = client.get_dependency_status("job-123")
print(status['reason'])
```

### Retries Exhausted

Check retry history:
```python
retry = client.get_retry_status("job-123")
for record in retry['history']:
    print(f"Attempt {record['attempt']}: {record['error']}")
```

### Outside Time Window

Check next available window:
```python
status = client.get_timewindow_status("job-123")
print(f"Next window starts: {status['next_window_start']}")
```

## License

LGPL-3.0-or-later

## See Also

- [Basic Scheduling](./SCHEDULING.md)
- [Job Management](./JOBS.md)
- [REST API Reference](../api/README.md)
