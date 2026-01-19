import { useState } from 'react';
import { Dashboard } from './components/Dashboard';
import { Login } from './components/Login';

function App() {
  // Set to true to skip login (no backend auth implemented yet)
  const [isAuthenticated, setIsAuthenticated] = useState(true);

  const handleLogin = async (username: string, password: string) => {
    // TODO: Implement actual authentication logic
    // For now, accept any non-empty credentials
    if (username && password) {
      setIsAuthenticated(true);
    } else {
      throw new Error('Invalid credentials');
    }
  };

  const handleLogout = () => {
    setIsAuthenticated(false);
  };

  if (!isAuthenticated) {
    return <Login onLogin={handleLogin} />;
  }

  return <Dashboard onLogout={handleLogout} />;
}

export default App;
