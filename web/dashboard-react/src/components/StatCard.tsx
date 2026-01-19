import React from 'react';

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: string;
  color?: string;
  trend?: {
    value: number;
    isPositive: boolean;
  };
}

export const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  subtitle,
  icon,
  color = '#3b82f6',
  trend,
}) => {
  return (
    <div
      style={{
        backgroundColor: '#fff',
        borderRadius: '4px',
        padding: '24px',
        border: '2px solid #e0e0e0',
        transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = '#f0583a';
        e.currentTarget.style.transform = 'translateY(-2px)';
        e.currentTarget.style.boxShadow = '0 8px 16px rgba(0, 0, 0, 0.1)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = '#e0e0e0';
        e.currentTarget.style.transform = 'translateY(0)';
        e.currentTarget.style.boxShadow = 'none';
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div style={{ flex: 1 }}>
          <div style={{
            fontSize: '12px',
            color: '#6b7280',
            marginBottom: '12px',
            fontWeight: '600',
            textTransform: 'uppercase',
            letterSpacing: '0.5px',
          }}>
            {title}
          </div>
          <div style={{ fontSize: '32px', fontWeight: '700', color: color, marginBottom: '4px' }}>
            {value}
          </div>
          {subtitle && (
            <div style={{ fontSize: '14px', color: '#6b7280', fontWeight: '500' }}>
              {subtitle}
            </div>
          )}
          {trend && (
            <div
              style={{
                fontSize: '12px',
                color: trend.isPositive ? '#10b981' : '#ef4444',
                marginTop: '8px',
                fontWeight: '600',
              }}
            >
              {trend.isPositive ? '↑' : '↓'} {Math.abs(trend.value)}%
            </div>
          )}
        </div>
        {icon && (
          <div style={{ fontSize: '40px', opacity: 0.5, color: color }}>
            {icon}
          </div>
        )}
      </div>
    </div>
  );
};
