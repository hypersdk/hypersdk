import React from 'react';

interface QuickLink {
  title: string;
  description: string;
  icon: string;
  href: string;
  onClick?: () => void;
}

interface QuickLinksProps {
  links: QuickLink[];
}

export const QuickLinks: React.FC<QuickLinksProps> = ({ links }) => {
  return (
    <div style={{
      backgroundColor: '#f0f2f7',
      padding: '48px 24px',
    }}>
      <div style={{
        maxWidth: '1400px',
        margin: '0 auto',
      }}>
        <h2 style={{
          margin: '0 0 32px 0',
          fontSize: '28px',
          fontWeight: '700',
          color: '#000',
          textTransform: 'uppercase',
          letterSpacing: '1px',
        }}>
          Quick Links
        </h2>

        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
          gap: '24px',
        }}>
          {links.map((link, index) => (
            <a
              key={index}
              href={link.href}
              onClick={(e) => {
                if (link.onClick) {
                  e.preventDefault();
                  link.onClick();
                }
              }}
              style={{
                backgroundColor: '#fff',
                border: '2px solid #e0e0e0',
                borderRadius: '4px',
                padding: '24px',
                textDecoration: 'none',
                transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
                display: 'block',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = '#f0583a';
                e.currentTarget.style.transform = 'translateY(-4px)';
                e.currentTarget.style.boxShadow = '0 8px 16px rgba(0, 0, 0, 0.1)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = '#e0e0e0';
                e.currentTarget.style.transform = 'translateY(0)';
                e.currentTarget.style.boxShadow = 'none';
              }}
            >
              <div style={{
                fontSize: '32px',
                marginBottom: '16px',
              }}>
                {link.icon}
              </div>
              <h3 style={{
                margin: '0 0 8px 0',
                fontSize: '18px',
                fontWeight: '700',
                color: '#000',
                textTransform: 'uppercase',
                letterSpacing: '0.5px',
              }}>
                {link.title}
              </h3>
              <p style={{
                margin: 0,
                fontSize: '14px',
                color: '#6b7280',
                lineHeight: 1.5,
              }}>
                {link.description}
              </p>
            </a>
          ))}
        </div>
      </div>
    </div>
  );
};
