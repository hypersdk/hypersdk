import React, { useEffect, useMemo, useState } from 'react';
import { Header } from './Header';
import { Hero } from './Hero';
import { QuickLinks } from './QuickLinks';
import { Footer } from './Footer';
import { StatCard } from './StatCard';
import { JobsTable } from './JobsTable';
import { AlertsList } from './AlertsList';
import { ChartContainer } from './ChartContainer';
import { JobSubmissionForm } from './JobSubmissionForm';
import { useWebSocket } from '../hooks/useWebSocket';
import { useMetricsHistory } from '../hooks/useMetricsHistory';
import { formatBytes, formatDuration, getStatusColor } from '../utils/formatters';
import { cancelJob, submitJob } from '../utils/api';

interface DashboardProps {
  onLogout?: () => void;
}

export const Dashboard: React.FC<DashboardProps> = ({ onLogout }) => {
  const wsUrl = `ws://${window.location.host}/ws`;
  const { data: metrics, connected, reconnecting, error } = useWebSocket({ url: wsUrl });
  const { history, addMetrics } = useMetricsHistory(60); // Keep last 60 data points
  const [showJobForm, setShowJobForm] = useState(false);

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

  const handleSubmitJob = async (jobData: unknown) => {
    try {
      await submitJob(jobData);
      setShowJobForm(false);
      alert('Job submitted successfully!');
    } catch (err) {
      console.error('Failed to submit job:', err);
      throw err;
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

  // Show dashboard even if WebSocket fails - just display a warning banner
  const hasConnectionIssue = error || (!metrics && !connected && !reconnecting);

  const quickLinks = [
    {
      title: 'My Jobs',
      description: 'View and manage all your export jobs',
      icon: '●',
      href: '#jobs',
      onClick: () => {},
    },
    {
      title: 'New Export',
      description: 'Start a new VM export job',
      icon: '◈',
      href: '#new-export',
      onClick: () => setShowJobForm(true),
    },
    {
      title: 'Job Status',
      description: 'Check the status of running jobs',
      icon: '◷',
      href: '#status',
      onClick: () => {},
    },
    {
      title: 'Providers',
      description: 'Manage cloud provider connections',
      icon: '▣',
      href: '#providers',
      onClick: () => {},
    },
  ];

  return (
    <div style={{ backgroundColor: '#f0f2f7', minHeight: '100vh' }}>
      <Header onLogout={onLogout} />

      <Hero
        title="Multi-Cloud VM Migration"
        subtitle="Seamlessly migrate and export virtual machines across vSphere, AWS, Azure, GCP, Hyper-V, and more"
        onNewJob={() => setShowJobForm(true)}
      />

      {/* Connection Status Bar */}
      {hasConnectionIssue && (
        <div style={{
          backgroundColor: '#fee2e2',
          borderBottom: '2px solid #ef4444',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
            padding: '16px 24px',
            textAlign: 'center',
          }}>
            <div style={{
              fontSize: '14px',
              fontWeight: '600',
              color: '#991b1b',
            }}>
              ⚠️ WebSocket Connection Failed - Real-time updates unavailable. The dashboard is running in demo mode.
            </div>
            {error && (
              <div style={{
                fontSize: '12px',
                color: '#991b1b',
                marginTop: '4px',
                opacity: 0.8,
              }}>
                Error: {error.message}
              </div>
            )}
          </div>
        </div>
      )}

      {connected && (
        <div style={{
          backgroundColor: '#10b98120',
          borderBottom: '2px solid #10b981',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
            padding: '12px 24px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <span
                style={{
                  width: '10px',
                  height: '10px',
                  borderRadius: '50%',
                  backgroundColor: '#10b981',
                }}
              />
              <span style={{
                fontSize: '14px',
                fontWeight: '600',
                color: '#10b981',
              }}>
                Connected
              </span>
            </div>
            {metrics && (
              <div style={{ fontSize: '14px', color: '#6b7280', fontWeight: '500' }}>
                Uptime: {formatDuration(metrics.uptime_seconds)}
              </div>
            )}
          </div>
        </div>
      )}

      <QuickLinks links={quickLinks} />

      {/* Job Submission Form */}
      {showJobForm && (
        <div style={{
          backgroundColor: '#f0f2f7',
          padding: '48px 24px',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
          }}>
            <div style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '24px',
            }}>
              <h2 style={{
                margin: 0,
                fontSize: '28px',
                fontWeight: '700',
                color: '#000',
                textTransform: 'uppercase',
                letterSpacing: '1px',
              }}>
                New Export Job
              </h2>
              <button
                onClick={() => setShowJobForm(false)}
                style={{
                  padding: '8px 16px',
                  backgroundColor: 'transparent',
                  color: '#222324',
                  border: '2px solid #222324',
                  borderRadius: '4px',
                  fontSize: '12px',
                  fontWeight: '700',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px',
                  cursor: 'pointer',
                  transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = '#f0583a';
                  e.currentTarget.style.color = '#f0583a';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = '#222324';
                  e.currentTarget.style.color = '#222324';
                }}
              >
                Close
              </button>
            </div>
            <JobSubmissionForm onSubmit={handleSubmitJob} />
          </div>
        </div>
      )}

      {/* Alerts */}
      {metrics?.alerts && metrics.alerts.length > 0 && (
        <div style={{
          backgroundColor: '#f0f2f7',
          padding: '24px',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
          }}>
            <AlertsList alerts={metrics.alerts} />
          </div>
        </div>
      )}

      {/* Stats Grid - Show demo data if no metrics */}
      {(metrics || hasConnectionIssue) && (
        <div style={{
          backgroundColor: '#f0f2f7',
          padding: '48px 24px',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
          }}>
            <h2 style={{
              margin: '0 0 24px 0',
              fontSize: '28px',
              fontWeight: '700',
              color: '#000',
              textTransform: 'uppercase',
              letterSpacing: '1px',
            }}>
              System Overview
            </h2>
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
            value={metrics?.jobs_active ?? 0}
            icon="●"
            color="#f0583a"
          />
          <StatCard
            title="Completed Jobs"
            value={metrics?.jobs_completed ?? 0}
            icon="✓"
            color="#10b981"
          />
          <StatCard
            title="Failed Jobs"
            value={metrics?.jobs_failed ?? 0}
            icon="✕"
            color="#ef4444"
          />
          <StatCard
            title="Queue Length"
            value={metrics?.queue_length ?? 0}
            subtitle={`${metrics?.jobs_pending ?? 0} pending`}
            icon="◷"
            color="#222324"
          />
          <StatCard
            title="Memory Usage"
            value={metrics ? formatBytes(metrics.memory_usage) : '0 B'}
            icon="▣"
            color="#222324"
          />
          <StatCard
            title="WebSocket Clients"
            value={metrics?.websocket_clients ?? 0}
            icon="◈"
            color="#f0583a"
          />
            </div>
          </div>
        </div>
      )}

      {/* Charts Row 1 */}
      <div style={{
        backgroundColor: '#f0f2f7',
        padding: '48px 24px',
      }}>
        <div style={{
          maxWidth: '1400px',
          margin: '0 auto',
        }}>
          <h2 style={{
            margin: '0 0 24px 0',
            fontSize: '28px',
            fontWeight: '700',
            color: '#000',
            textTransform: 'uppercase',
            letterSpacing: '1px',
          }}>
            Performance Metrics
          </h2>
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
        </div>
      </div>

      {/* Charts Row 2 */}
      {providerChartData.length > 0 && (
        <div style={{
          backgroundColor: '#f0f2f7',
          padding: '48px 24px',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
          }}>
            <ChartContainer
              title="Jobs by Provider"
              type="pie"
              data={providerChartData}
            />
          </div>
        </div>
      )}

      {/* Jobs Table */}
      {metrics?.recent_jobs && (
        <div style={{
          backgroundColor: '#f0f2f7',
          padding: '48px 24px',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
          }}>
            <h2 style={{
              margin: '0 0 24px 0',
              fontSize: '28px',
              fontWeight: '700',
              color: '#000',
              textTransform: 'uppercase',
              letterSpacing: '1px',
            }}>
              Recent Jobs
            </h2>
            <JobsTable jobs={metrics.recent_jobs} onCancelJob={handleCancelJob} />
          </div>
        </div>
      )}

      {/* System Info */}
      {metrics && (
        <div style={{
          backgroundColor: '#f0f2f7',
          padding: '48px 24px',
        }}>
          <div style={{
            maxWidth: '1400px',
            margin: '0 auto',
          }}>
            <h2 style={{
              margin: '0 0 24px 0',
              fontSize: '28px',
              fontWeight: '700',
              color: '#000',
              textTransform: 'uppercase',
              letterSpacing: '1px',
            }}>
              System Health
            </h2>
            <div
              style={{
                backgroundColor: '#fff',
                borderRadius: '8px',
                padding: '24px',
                border: '2px solid #e0e0e0',
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                gap: '24px',
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
          </div>
        </div>
      )}

      <Footer />
    </div>
  );
};
