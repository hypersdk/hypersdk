import React, { useState, useMemo } from 'react';
import type { JobInfo } from '../types/metrics';
import { formatDuration, formatRelativeTime, getStatusColor, getStatusIcon } from '../utils/formatters';

interface JobsTableProps {
  jobs: JobInfo[];
  onCancelJob?: (jobId: string) => void;
}

type SortField = keyof JobInfo | 'none';
type SortDirection = 'asc' | 'desc';

export const JobsTable: React.FC<JobsTableProps> = ({ jobs, onCancelJob }) => {
  const [sortField, setSortField] = useState<SortField>('start_time');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
  const [filterStatus, setFilterStatus] = useState<string>('all');

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const filteredAndSortedJobs = useMemo(() => {
    let filtered = jobs;

    if (filterStatus !== 'all') {
      filtered = jobs.filter((job) => job.status === filterStatus);
    }

    if (sortField !== 'none') {
      filtered = [...filtered].sort((a, b) => {
        const aVal = a[sortField];
        const bVal = b[sortField];

        if (aVal === undefined || aVal === null) return 1;
        if (bVal === undefined || bVal === null) return -1;

        if (typeof aVal === 'string' && typeof bVal === 'string') {
          return sortDirection === 'asc'
            ? aVal.localeCompare(bVal)
            : bVal.localeCompare(aVal);
        }

        if (typeof aVal === 'number' && typeof bVal === 'number') {
          return sortDirection === 'asc' ? aVal - bVal : bVal - aVal;
        }

        return 0;
      });
    }

    return filtered;
  }, [jobs, sortField, sortDirection, filterStatus]);

  const tableHeaderStyle: React.CSSProperties = {
    padding: '16px',
    textAlign: 'left',
    fontSize: '12px',
    fontWeight: '700',
    color: '#000',
    backgroundColor: '#f0f2f7',
    borderBottom: '2px solid #e0e0e0',
    cursor: 'pointer',
    userSelect: 'none',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
  };

  const tableCellStyle: React.CSSProperties = {
    padding: '16px',
    fontSize: '14px',
    borderBottom: '1px solid #e0e0e0',
  };

  return (
    <div style={{ backgroundColor: '#fff', borderRadius: '4px', border: '2px solid #e0e0e0' }}>
      <div style={{ padding: '20px', borderBottom: '2px solid #e0e0e0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
          <div style={{
            width: '4px',
            height: '24px',
            backgroundColor: '#f0583a',
          }} />
          <h3 style={{ margin: 0, fontSize: '18px', fontWeight: '700', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Filter Jobs</h3>
        </div>
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value)}
          style={{
            padding: '10px 16px',
            borderRadius: '4px',
            border: '2px solid #222324',
            fontSize: '14px',
            fontWeight: '600',
            backgroundColor: '#fff',
            cursor: 'pointer',
            transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
          }}
          onFocus={(e) => e.currentTarget.style.borderColor = '#f0583a'}
          onBlur={(e) => e.currentTarget.style.borderColor = '#222324'}
        >
          <option value="all">All Status</option>
          <option value="pending">Pending</option>
          <option value="running">Running</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={tableHeaderStyle} onClick={() => handleSort('status')}>
                Status {sortField === 'status' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th style={tableHeaderStyle} onClick={() => handleSort('name')}>
                Name {sortField === 'name' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th style={tableHeaderStyle} onClick={() => handleSort('vm_name')}>
                VM {sortField === 'vm_name' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th style={tableHeaderStyle} onClick={() => handleSort('provider')}>
                Provider {sortField === 'provider' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th style={tableHeaderStyle} onClick={() => handleSort('progress')}>
                Progress {sortField === 'progress' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th style={tableHeaderStyle} onClick={() => handleSort('duration_seconds')}>
                Duration {sortField === 'duration_seconds' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th style={tableHeaderStyle} onClick={() => handleSort('start_time')}>
                Started {sortField === 'start_time' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th style={tableHeaderStyle}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredAndSortedJobs.length === 0 ? (
              <tr>
                <td colSpan={8} style={{ ...tableCellStyle, textAlign: 'center', color: '#9ca3af' }}>
                  No jobs found
                </td>
              </tr>
            ) : (
              filteredAndSortedJobs.map((job) => (
                <tr key={job.id} style={{ transition: 'background-color 0.2s' }}>
                  <td style={tableCellStyle}>
                    <span
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        padding: '4px 12px',
                        borderRadius: '12px',
                        fontSize: '12px',
                        fontWeight: '500',
                        backgroundColor: getStatusColor(job.status) + '20',
                        color: getStatusColor(job.status),
                      }}
                    >
                      {getStatusIcon(job.status)} {job.status}
                    </span>
                  </td>
                  <td style={tableCellStyle}>
                    <div style={{ fontWeight: '500' }}>{job.name}</div>
                    <div style={{ fontSize: '12px', color: '#6b7280' }}>{job.id.substring(0, 8)}</div>
                  </td>
                  <td style={tableCellStyle}>
                    <div>{job.vm_name}</div>
                    {job.format && (
                      <div style={{ fontSize: '12px', color: '#6b7280' }}>
                        {job.format.toUpperCase()}
                        {job.compress && ' • Compressed'}
                      </div>
                    )}
                  </td>
                  <td style={tableCellStyle}>{job.provider || 'N/A'}</td>
                  <td style={tableCellStyle}>
                    <div style={{ width: '100%', backgroundColor: '#e5e7eb', borderRadius: '4px', height: '8px' }}>
                      <div
                        style={{
                          width: `${job.progress}%`,
                          backgroundColor: getStatusColor(job.status),
                          borderRadius: '4px',
                          height: '100%',
                          transition: 'width 0.3s',
                        }}
                      />
                    </div>
                    <div style={{ fontSize: '12px', color: '#6b7280', marginTop: '4px' }}>
                      {job.progress}%
                    </div>
                  </td>
                  <td style={tableCellStyle}>{formatDuration(job.duration_seconds)}</td>
                  <td style={tableCellStyle}>
                    <div style={{ fontSize: '12px', color: '#6b7280' }}>
                      {formatRelativeTime(job.start_time)}
                    </div>
                  </td>
                  <td style={tableCellStyle}>
                    {job.status === 'running' && onCancelJob && (
                      <button
                        onClick={() => onCancelJob(job.id)}
                        style={{
                          padding: '4px 12px',
                          fontSize: '12px',
                          borderRadius: '4px',
                          border: '1px solid #ef4444',
                          backgroundColor: '#fff',
                          color: '#ef4444',
                          cursor: 'pointer',
                        }}
                      >
                        Cancel
                      </button>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};
