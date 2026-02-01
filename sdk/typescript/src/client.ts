/**
 * HyperSDK TypeScript client
 */

import {
  Job,
  JobDefinition,
  JobStatus,
  JobProgress,
  QueryRequest,
  QueryResponse,
  SubmitResponse,
  CancelRequest,
  CancelResponse,
  DaemonStatus,
  ScheduledJob,
  Webhook,
  HealthResponse,
  CapabilitiesResponse,
} from './models';

import {
  HyperSDKError,
  AuthenticationError,
  JobNotFoundError,
  APIError,
} from './errors';

export interface HyperSDKConfig {
  baseURL: string;
  apiKey?: string;
  timeout?: number;
  headers?: Record<string, string>;
}

export class HyperSDK {
  private baseURL: string;
  private apiKey?: string;
  private timeout: number;
  private headers: Record<string, string>;
  private token?: string;

  constructor(config: HyperSDKConfig | string) {
    if (typeof config === 'string') {
      this.baseURL = config.replace(/\/$/, '');
      this.timeout = 30000;
      this.headers = {};
    } else {
      this.baseURL = config.baseURL.replace(/\/$/, '');
      this.apiKey = config.apiKey;
      this.timeout = config.timeout || 30000;
      this.headers = config.headers || {};
    }

    if (this.apiKey) {
      this.headers['X-API-Key'] = this.apiKey;
    }
  }

  private buildURL(path: string): string {
    return `${this.baseURL}${path}`;
  }

  private async request<T>(
    method: string,
    path: string,
    options: {
      body?: any;
      params?: Record<string, string>;
      headers?: Record<string, string>;
    } = {}
  ): Promise<T> {
    const url = new URL(this.buildURL(path));

    if (options.params) {
      Object.entries(options.params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
      });
    }

    const headers: Record<string, string> = {
      ...this.headers,
      ...options.headers,
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    if (options.body) {
      headers['Content-Type'] = 'application/json';
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url.toString(), {
        method,
        headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (response.status === 404) {
        throw new JobNotFoundError(`Resource not found: ${path}`);
      }

      if (response.status === 401) {
        throw new AuthenticationError('Authentication failed');
      }

      if (!response.ok) {
        let errorMessage = response.statusText;
        let errorData: any;

        try {
          errorData = await response.json();
          errorMessage = errorData.error || errorMessage;
        } catch (e) {
          // Ignore JSON parse errors
        }

        throw new APIError(
          `API error: ${errorMessage}`,
          response.status,
          errorData
        );
      }

      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      } else if (contentType && contentType.includes('text/')) {
        return (await response.text()) as any;
      }

      return null as any;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof HyperSDKError) {
        throw error;
      }

      if (error instanceof Error && error.name === 'AbortError') {
        throw new APIError('Request timeout');
      }

      throw new APIError(`Request failed: ${error}`);
    }
  }

  // Authentication

  async login(username: string, password: string): Promise<string> {
    const response = await this.request<{ token: string; expires_at: string }>(
      'POST',
      '/api/login',
      {
        body: { username, password },
      }
    );
    this.token = response.token;
    return this.token;
  }

  async logout(): Promise<void> {
    await this.request('POST', '/api/logout');
    this.token = undefined;
  }

  // Health & Status

  async health(): Promise<HealthResponse> {
    return this.request<HealthResponse>('GET', '/health');
  }

  async status(): Promise<DaemonStatus> {
    return this.request<DaemonStatus>('GET', '/status');
  }

  async capabilities(): Promise<CapabilitiesResponse> {
    return this.request<CapabilitiesResponse>('GET', '/capabilities');
  }

  // Job Management

  async submitJob(jobDef: JobDefinition): Promise<string> {
    const response = await this.request<SubmitResponse>('POST', '/jobs/submit', {
      body: jobDef,
    });

    if (response.accepted === 0) {
      const error = response.errors?.[0] || 'Unknown error';
      throw new APIError(`Job submission failed: ${error}`);
    }

    return response.job_ids[0];
  }

  async submitJobs(jobDefs: JobDefinition[]): Promise<string[]> {
    const response = await this.request<SubmitResponse>('POST', '/jobs/submit', {
      body: jobDefs,
    });
    return response.job_ids;
  }

  async getJob(jobId: string): Promise<Job> {
    return this.request<Job>('GET', `/jobs/${jobId}`);
  }

  async queryJobs(query: QueryRequest = {}): Promise<Job[]> {
    const response = await this.request<QueryResponse>('POST', '/jobs/query', {
      body: query,
    });
    return response.jobs;
  }

  async listJobs(all: boolean = true): Promise<Job[]> {
    const response = await this.request<QueryResponse>('GET', '/jobs/query', {
      params: all ? { all: 'true' } : {},
    });
    return response.jobs;
  }

  async cancelJob(jobId: string): Promise<boolean> {
    const response = await this.request<CancelResponse>('POST', '/jobs/cancel', {
      body: { job_ids: [jobId] },
    });

    if (response.cancelled.includes(jobId)) {
      return true;
    }

    if (response.failed.includes(jobId)) {
      const error = response.errors?.[jobId] || 'Unknown error';
      throw new APIError(`Failed to cancel job: ${error}`);
    }

    return false;
  }

  async cancelJobs(jobIds: string[]): Promise<CancelResponse> {
    return this.request<CancelResponse>('POST', '/jobs/cancel', {
      body: { job_ids: jobIds },
    });
  }

  async getJobProgress(jobId: string): Promise<JobProgress> {
    return this.request<JobProgress>('GET', `/jobs/progress/${jobId}`);
  }

  async getJobLogs(jobId: string): Promise<string> {
    return this.request<string>('GET', `/jobs/logs/${jobId}`);
  }

  async getJobETA(jobId: string): Promise<string> {
    const response = await this.request<{ eta?: string; estimated_remaining?: string }>(
      'GET',
      `/jobs/eta/${jobId}`
    );
    return response.estimated_remaining || response.eta || 'Unknown';
  }

  // VM Operations

  async listVMs(vcenterConfig: any): Promise<any[]> {
    return this.request<any[]>('POST', '/vms/list', {
      body: vcenterConfig,
    });
  }

  async getVMInfo(vcenterConfig: any, vmPath: string): Promise<any> {
    return this.request<any>('POST', '/vms/info', {
      body: { vcenter: vcenterConfig, vm_path: vmPath },
    });
  }

  async shutdownVM(vcenterConfig: any, vmPath: string): Promise<any> {
    return this.request<any>('POST', '/vms/shutdown', {
      body: { vcenter: vcenterConfig, vm_path: vmPath },
    });
  }

  // Schedule Management

  async listSchedules(): Promise<ScheduledJob[]> {
    return this.request<ScheduledJob[]>('GET', '/schedules');
  }

  async createSchedule(schedule: ScheduledJob): Promise<ScheduledJob> {
    return this.request<ScheduledJob>('POST', '/schedules', {
      body: schedule,
    });
  }

  async getSchedule(scheduleId: string): Promise<ScheduledJob> {
    return this.request<ScheduledJob>('GET', `/schedules/${scheduleId}`);
  }

  async updateSchedule(
    scheduleId: string,
    schedule: ScheduledJob
  ): Promise<ScheduledJob> {
    return this.request<ScheduledJob>('PUT', `/schedules/${scheduleId}`, {
      body: schedule,
    });
  }

  async deleteSchedule(scheduleId: string): Promise<void> {
    await this.request('DELETE', `/schedules/${scheduleId}`);
  }

  async enableSchedule(scheduleId: string): Promise<void> {
    await this.request('POST', `/schedules/${scheduleId}/enable`);
  }

  async disableSchedule(scheduleId: string): Promise<void> {
    await this.request('POST', `/schedules/${scheduleId}/disable`);
  }

  async triggerSchedule(scheduleId: string): Promise<void> {
    await this.request('POST', `/schedules/${scheduleId}/trigger`);
  }

  // Webhook Management

  async listWebhooks(): Promise<Webhook[]> {
    return this.request<Webhook[]>('GET', '/webhooks');
  }

  async addWebhook(webhook: Webhook): Promise<void> {
    await this.request('POST', '/webhooks', { body: webhook });
  }

  async testWebhook(url: string): Promise<void> {
    await this.request('POST', '/webhooks/test', { body: { url } });
  }

  async deleteWebhook(webhookId: string): Promise<void> {
    await this.request('DELETE', `/webhooks/${webhookId}`);
  }

  // Libvirt Operations

  async listDomains(): Promise<any[]> {
    return this.request<any[]>('GET', '/libvirt/domains');
  }

  async getDomain(name: string): Promise<any> {
    return this.request<any>('GET', '/libvirt/domain', {
      params: { name },
    });
  }

  async startDomain(name: string): Promise<void> {
    await this.request('POST', '/libvirt/domain/start', {
      body: { name },
    });
  }

  async shutdownDomain(name: string): Promise<void> {
    await this.request('POST', '/libvirt/domain/shutdown', {
      body: { name },
    });
  }

  async listSnapshots(domain: string): Promise<any[]> {
    return this.request<any[]>('GET', '/libvirt/snapshots', {
      params: { domain },
    });
  }

  async createSnapshot(
    domain: string,
    name: string,
    description?: string
  ): Promise<void> {
    await this.request('POST', '/libvirt/snapshot/create', {
      body: { domain, name, description },
    });
  }

  // Hyper2KVM Integration

  async convertVM(sourcePath: string, outputPath: string): Promise<string> {
    const response = await this.request<{ conversion_id: string }>(
      'POST',
      '/convert/vm',
      {
        body: { source_path: sourcePath, output_path: outputPath },
      }
    );
    return response.conversion_id;
  }

  async getConversionStatus(conversionId: string): Promise<any> {
    return this.request<any>('GET', '/convert/status', {
      params: { conversion_id: conversionId },
    });
  }
}

export default HyperSDK;
