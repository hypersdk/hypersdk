import React, { useEffect, useMemo } from 'react';
import { StatCard } from './StatCard';
import { JobsTable } from './JobsTable';
import { AlertsList } from './AlertsList';
import { ChartContainer } from './ChartContainer';
import { useWebSocket } from '../hooks/useWebSocket';
import { useMetricsHistory } from '../hooks/useMetricsHistory';
import { formatBytes, formatDuration, getStatusColor } from '../utils/formatters';
import { cancelJob } from '../utils/api';

export const Dashboard: React.FC = () => {
  const wsUrl = `ws://${window.location.host}/ws`;
  const { data: metrics, connected, reconnecting, error } = useWebSocket({ url: wsUrl });
  const { history, addMetrics } = useMetricsHistory(60); // Keep last 60 data points

  useEffect(() => {
    if (metrics) {
      addMetrics(metrics);
    }
  }, [metrics, addMetrics]);

  const handleCancelJob = async (jobId: string) => {
    try {
      await cancelJob(jobId);
    } catch (err) {
      console.error('Failed to cancel job:', err);
      alert('Failed to cancel job');
    }
  };

  const jobsChartData = useMemo(() => {
    return history.map((m) => ({
      timestamp: m.timestamp,
      Active: m.jobs_active,
      Completed: m.jobs_completed,
      Failed: m.jobs_failed,
      Pending: m.jobs_pending,
    }));
  }, [history]);

  const resourceChartData = useMemo(() => {
    return history.map((m) => ({
      timestamp: m.timestamp,
      'Memory (MB)': Math.round(m.memory_usage / 1024 / 1024),
      'CPU (%)': m.cpu_usage,
      Goroutines: m.goroutines,
    }));
  }, [history]);

  const providerChartData = useMemo(() => {
    if (!metrics?.provider_stats) return [];

    return Object.entries(metrics.provider_stats).map(([name, stats]) => ({
      name,
      value: stats.jobs_total,
      completed: stats.jobs_completed,
      failed: stats.jobs_failed,
    }));
  }, [metrics?.provider_stats]);

  if (error) {
    return (
      <div
        style={{
          padding: '40px',
          textAlign: 'center',
          backgroundColor: '#fee2e2',
          color: '#991b1b',
          borderRadius: '8px',
          margin: '20px',
        }}
      >
        <h2>WebSocket Connection Error</h2>
        <p>{error.message}</p>
      </div>
    );
  }

  if (!metrics && !reconnecting) {
    return (
      <div
        style={{
          padding: '40px',
          textAlign: 'center',
          color: '#6b7280',
        }}
      >
        <h2>Connecting to HyperSDK...</h2>
      </div>
    );
  }

  return (
    <div style={{ padding: '20px', backgroundColor: '#f9fafb', minHeight: '100vh' }}>
      {/* Header */}
      <div style={{ marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1 style={{ margin: 0, fontSize: '28px', fontWeight: 'bold', color: '#111827' }}>
            HyperSDK Dashboard
          </h1>
          <p style={{ margin: '4px 0 0 0', color: '#6b7280' }}>
            Multi-Cloud VM Migration & Management
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              padding: '6px 12px',
              borderRadius: '12px',
              fontSize: '12px',
              fontWeight: '500',
              backgroundColor: connected ? '#10b98120' : reconnecting ? '#f59e0b20' : '#ef444420',
              color: connected ? '#10b981' : reconnecting ? '#f59e0b' : '#ef4444',
            }}
          >
            <span
              style={{
                width: '8px',
                height: '8px',
                borderRadius: '50%',
                backgroundColor: connected ? '#10b981' : reconnecting ? '#f59e0b' : '#ef4444',
                marginRight: '6px',
              }}
            />
            {connected ? 'Connected' : reconnecting ? 'Reconnecting...' : 'Disconnected'}
          </div>
          {metrics && (
            <div style={{ fontSize: '12px', color: '#6b7280' }}>
              Uptime: {formatDuration(metrics.uptime_seconds)}
            </div>
          )}
        </div>
      </div>

      {/* Alerts */}
      {metrics?.alerts && metrics.alerts.length > 0 && (
        <div style={{ marginBottom: '24px' }}>
          <AlertsList alerts={metrics.alerts} />
        </div>
      )}

      {/* Stats Grid */}
      {metrics && (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
            gap: '16px',
            marginBottom: '24px',
          }}
        >
          <StatCard
            title="Active Jobs"
            value={metrics.jobs_active}
            icon="▶️"
            color="#3b82f6"
          />
          <StatCard
            title="Completed Jobs"
            value={metrics.jobs_completed}
            icon="✅"
            color="#10b981"
          />
          <StatCard
            title="Failed Jobs"
            value={metrics.jobs_failed}
            icon="❌"
            color="#ef4444"
          />
          <StatCard
            title="Queue Length"
            value={metrics.queue_length}
            subtitle={`${metrics.jobs_pending} pending`}
            icon="⏳"
            color="#f59e0b"
          />
          <StatCard
            title="Memory Usage"
            value={formatBytes(metrics.memory_usage)}
            icon="💾"
            color="#8b5cf6"
          />
          <StatCard
            title="WebSocket Clients"
            value={metrics.websocket_clients}
            icon="🔌"
            color="#ec4899"
          />
        </div>
      )}

      {/* Charts Row 1 */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
          gap: '16px',
          marginBottom: '24px',
        }}
      >
        <ChartContainer
          title="Job Activity Over Time"
          type="line"
          data={jobsChartData}
          dataKeys={['Active', 'Completed', 'Failed', 'Pending']}
          colors={['#3b82f6', '#10b981', '#ef4444', '#f59e0b']}
        />
        <ChartContainer
          title="System Resources"
          type="line"
          data={resourceChartData}
          dataKeys={['Memory (MB)', 'CPU (%)', 'Goroutines']}
          colors={['#8b5cf6', '#ec4899', '#14b8a6']}
        />
      </div>

      {/* Charts Row 2 */}
      {providerChartData.length > 0 && (
        <div style={{ marginBottom: '24px' }}>
          <ChartContainer
            title="Jobs by Provider"
            type="pie"
            data={providerChartData}
          />
        </div>
      )}

      {/* Jobs Table */}
      {metrics?.recent_jobs && (
        <div style={{ marginBottom: '24px' }}>
          <JobsTable jobs={metrics.recent_jobs} onCancelJob={handleCancelJob} />
        </div>
      )}

      {/* System Info */}
      {metrics && (
        <div
          style={{
            backgroundColor: '#fff',
            borderRadius: '8px',
            padding: '16px',
            boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
            gap: '16px',
          }}
        >
          <div>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>
              System Health
            </div>
            <div style={{ fontSize: '16px', fontWeight: '600', color: getStatusColor(metrics.system_health) }}>
              {metrics.system_health.toUpperCase()}
            </div>
          </div>
          <div>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>
              HTTP Requests
            </div>
            <div style={{ fontSize: '16px', fontWeight: '600' }}>
              {metrics.http_requests.toLocaleString()}
            </div>
          </div>
          <div>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>
              HTTP Errors
            </div>
            <div style={{ fontSize: '16px', fontWeight: '600', color: metrics.http_errors > 0 ? '#ef4444' : '#10b981' }}>
              {metrics.http_errors.toLocaleString()}
            </div>
          </div>
          <div>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>
              Avg Response Time
            </div>
            <div style={{ fontSize: '16px', fontWeight: '600' }}>
              {metrics.avg_response_time.toFixed(2)}ms
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
