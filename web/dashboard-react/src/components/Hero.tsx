import React from 'react';

interface HeroProps {
  title: string;
  subtitle: string;
  onNewJob?: () => void;
}

export const Hero: React.FC<HeroProps> = ({ title, subtitle, onNewJob }) => {
  return (
    <div style={{
      backgroundColor: '#f0583a',
      padding: '64px 24px',
      textAlign: 'center',
      position: 'relative',
      overflow: 'hidden',
    }}>
      {/* Background Pattern */}
      <div style={{
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        opacity: 0.1,
        backgroundImage: 'linear-gradient(45deg, #000 25%, transparent 25%, transparent 75%, #000 75%, #000), linear-gradient(45deg, #000 25%, transparent 25%, transparent 75%, #000 75%, #000)',
        backgroundSize: '60px 60px',
        backgroundPosition: '0 0, 30px 30px',
      }} />

      <div style={{
        maxWidth: '1400px',
        margin: '0 auto',
        position: 'relative',
        zIndex: 1,
      }}>
        <h1 style={{
          margin: '0 0 16px 0',
          fontSize: '48px',
          fontWeight: '700',
          color: '#fff',
          textTransform: 'uppercase',
          letterSpacing: '2px',
          lineHeight: 1.2,
        }}>
          {title}
        </h1>
        <p style={{
          margin: '0 0 32px 0',
          fontSize: '20px',
          color: '#fff',
          opacity: 0.95,
          maxWidth: '800px',
          marginLeft: 'auto',
          marginRight: 'auto',
        }}>
          {subtitle}
        </p>

        {onNewJob && (
          <button
            onClick={onNewJob}
            style={{
              padding: '16px 48px',
              backgroundColor: '#fff',
              color: '#f0583a',
              border: '3px solid #fff',
              borderRadius: '4px',
              fontSize: '16px',
              fontWeight: '700',
              textTransform: 'uppercase',
              letterSpacing: '1px',
              cursor: 'pointer',
              transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
              boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = '#000';
              e.currentTarget.style.borderColor = '#000';
              e.currentTarget.style.color = '#fff';
              e.currentTarget.style.transform = 'translateY(-2px)';
              e.currentTarget.style.boxShadow = '0 6px 16px rgba(0, 0, 0, 0.25)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = '#fff';
              e.currentTarget.style.borderColor = '#fff';
              e.currentTarget.style.color = '#f0583a';
              e.currentTarget.style.transform = 'translateY(0)';
              e.currentTarget.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.15)';
            }}
          >
            New Export Job
          </button>
        )}
      </div>
    </div>
  );
};
