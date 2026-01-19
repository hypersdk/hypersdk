import React, { useState } from 'react';

interface LoginProps {
  onLogin: (username: string, password: string) => Promise<void>;
}

export const Login: React.FC<LoginProps> = ({ onLogin }) => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      await onLogin(username, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setIsLoading(false);
    }
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '16px 20px',
    border: '2px solid #000',
    borderRadius: '4px',
    fontSize: '16px',
    backgroundColor: '#fff',
    color: '#000',
    transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
  };

  const labelStyle: React.CSSProperties = {
    display: 'block',
    fontSize: '14px',
    fontWeight: '600',
    color: '#000',
    marginBottom: '10px',
  };

  return (
    <div style={{
      minHeight: '100vh',
      backgroundColor: '#f0f2f7',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '20px',
    }}>
      <div style={{
        backgroundColor: '#fff',
        borderRadius: '4px',
        padding: '48px 40px',
        width: '100%',
        maxWidth: '480px',
        border: '1px solid #e0e0e0',
      }}>
        {/* Logo/Title */}
        <div style={{ textAlign: 'center', marginBottom: '40px' }}>
          <h1 style={{
            fontSize: '32px',
            fontWeight: '700',
            color: '#000',
            margin: '0 0 8px 0',
          }}>
            HyperSDK
          </h1>
          <p style={{
            fontSize: '16px',
            color: '#222324',
            margin: 0,
          }}>
            Sign in to your account
          </p>
        </div>

        {/* Error Message */}
        {error && (
          <div style={{
            padding: '16px',
            backgroundColor: '#fee2e2',
            color: '#991b1b',
            borderRadius: '4px',
            marginBottom: '24px',
            fontSize: '14px',
            fontWeight: '500',
          }}>
            {error}
          </div>
        )}

        {/* Login Form */}
        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom: '24px' }}>
            <label htmlFor="username" style={labelStyle}>
              Username
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              disabled={isLoading}
              style={inputStyle}
              onFocus={(e) => {
                e.currentTarget.style.borderColor = '#f0583a';
                e.currentTarget.style.outline = 'none';
              }}
              onBlur={(e) => {
                e.currentTarget.style.borderColor = '#000';
              }}
            />
          </div>

          <div style={{ marginBottom: '32px' }}>
            <label htmlFor="password" style={labelStyle}>
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              disabled={isLoading}
              style={inputStyle}
              onFocus={(e) => {
                e.currentTarget.style.borderColor = '#f0583a';
                e.currentTarget.style.outline = 'none';
              }}
              onBlur={(e) => {
                e.currentTarget.style.borderColor = '#000';
              }}
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            onMouseEnter={(e) => {
              if (!isLoading) {
                e.currentTarget.style.backgroundColor = '#d94b32';
                e.currentTarget.style.transform = 'translateY(-2px)';
              }
            }}
            onMouseLeave={(e) => {
              if (!isLoading) {
                e.currentTarget.style.backgroundColor = '#f0583a';
                e.currentTarget.style.transform = 'translateY(0)';
              }
            }}
            style={{
              width: '100%',
              padding: '16px',
              backgroundColor: isLoading ? '#9ca3af' : '#f0583a',
              color: '#fff',
              border: 'none',
              borderRadius: '4px',
              fontSize: '16px',
              fontWeight: '600',
              cursor: isLoading ? 'not-allowed' : 'pointer',
              transition: 'all 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
            }}
          >
            {isLoading ? 'signing in...' : 'sign in'}
          </button>
        </form>

        {/* Footer Links */}
        <div style={{
          marginTop: '32px',
          paddingTop: '24px',
          borderTop: '1px solid #e0e0e0',
          textAlign: 'center',
        }}>
          <a
            href="#"
            style={{
              color: '#222324',
              fontSize: '14px',
              textDecoration: 'none',
              transition: 'color 0.25s cubic-bezier(0.215, 0.61, 0.355, 1)',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = '#f0583a';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = '#222324';
            }}
          >
            Forgot password?
          </a>
        </div>
      </div>
    </div>
  );
};
