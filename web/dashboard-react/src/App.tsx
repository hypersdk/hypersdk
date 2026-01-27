import { useState, useEffect } from 'react';
import { Dashboard } from './components/Dashboard';
import { Login } from './components/Login';
import VSphereExportWorkflow from './components/VSphereExportWorkflow';

type AppView = 'dashboard' | 'export' | 'jobs' | 'workflows';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [currentView, setCurrentView] = useState<AppView>('dashboard');
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);

  // Check for saved credentials on mount
  useEffect(() => {
    const savedUsername = localStorage.getItem('hypersdk_username');
    const savedPassword = localStorage.getItem('hypersdk_password');
    const rememberMe = localStorage.getItem('hypersdk_remember') === 'true';

    if (rememberMe && savedUsername && savedPassword) {
      // Auto-login with saved credentials
      setIsAuthenticated(true);
    }
    setIsCheckingAuth(false);
  }, []);

  const handleLogin = async (username: string, password: string) => {
    // Simple authentication - accept any non-empty credentials
    if (username && password) {
      setIsAuthenticated(true);
      console.log('User logged in:', username);
    } else {
      throw new Error('Please enter username and password');
    }
  };

  const handleLogout = () => {
    setIsAuthenticated(false);
    setCurrentView('dashboard');
    // Clear saved credentials on logout
    localStorage.removeItem('hypersdk_username');
    localStorage.removeItem('hypersdk_password');
    localStorage.removeItem('hypersdk_remember');
    console.log('User logged out');
  };

  // Show loading while checking authentication
  if (isCheckingAuth) {
    return (
      <div style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: '#f0f2f7',
      }}>
        <div style={{
          fontSize: '18px',
          color: '#222324',
          fontWeight: '600',
        }}>
          Loading...
        </div>
      </div>
    );
  }

  // SHOW LOGIN SCREEN if not authenticated
  if (!isAuthenticated) {
    return <Login onLogin={handleLogin} />;
  }

  // AFTER LOGIN - SHOW DASHBOARD WITH NAVIGATION
  const renderNavigation = () => (
    <nav style={styles.navbar}>
      <div style={styles.navbarBrand}>
        <h1 style={styles.brandTitle}>🚀 HyperSDK</h1>
        <span style={styles.brandSubtitle}>Multi-Cloud VM Export Platform</span>
      </div>
      <div style={styles.navbarMenu}>
        <button
          onClick={() => setCurrentView('dashboard')}
          style={{
            ...styles.navButton,
            ...(currentView === 'dashboard' ? styles.navButtonActive : {})
          }}
        >
          📊 Dashboard
        </button>
        <button
          onClick={() => setCurrentView('export')}
          style={{
            ...styles.navButton,
            ...(currentView === 'export' ? styles.navButtonActive : {})
          }}
        >
          📤 Export Workflow
        </button>
        <button
          onClick={() => setCurrentView('jobs')}
          style={{
            ...styles.navButton,
            ...(currentView === 'jobs' ? styles.navButtonActive : {})
          }}
        >
          📋 Jobs
        </button>
        <button
          onClick={() => setCurrentView('workflows')}
          style={{
            ...styles.navButton,
            ...(currentView === 'workflows' ? styles.navButtonActive : {})
          }}
        >
          ⚙️ Workflows
        </button>
        <button onClick={handleLogout} style={styles.logoutButton}>
          🚪 Logout
        </button>
      </div>
    </nav>
  );

  const renderContent = () => {
    switch (currentView) {
      case 'export':
        return <VSphereExportWorkflow />;
      case 'jobs':
      case 'workflows':
      case 'dashboard':
      default:
        return <Dashboard onLogout={handleLogout} />;
    }
  };

  return (
    <div style={styles.app}>
      {renderNavigation()}
      <main style={styles.main}>
        {renderContent()}
      </main>
    </div>
  );
}

const styles: { [key: string]: React.CSSProperties } = {
  app: {
    minHeight: '100vh',
    backgroundColor: '#f0f2f7',
  },
  navbar: {
    backgroundColor: '#222324',
    color: '#fff',
    padding: '15px 30px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    boxShadow: '0 2px 4px rgba(0,0,0,0.1)',
  },
  navbarBrand: {
    display: 'flex',
    flexDirection: 'column',
  },
  brandTitle: {
    margin: 0,
    fontSize: '24px',
    fontWeight: 'bold',
  },
  brandSubtitle: {
    fontSize: '12px',
    color: '#ccc',
  },
  navbarMenu: {
    display: 'flex',
    gap: '10px',
    alignItems: 'center',
  },
  navButton: {
    padding: '10px 20px',
    backgroundColor: 'transparent',
    color: '#fff',
    border: '2px solid transparent',
    borderRadius: '5px',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 'bold',
    transition: 'all 0.3s',
  },
  navButtonActive: {
    backgroundColor: '#f0583a',
    borderColor: '#f0583a',
  },
  logoutButton: {
    padding: '10px 20px',
    backgroundColor: '#f44336',
    color: '#fff',
    border: 'none',
    borderRadius: '5px',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 'bold',
    marginLeft: '20px',
  },
  main: {
    minHeight: 'calc(100vh - 80px)',
  },
};

export default App;
