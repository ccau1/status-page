import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router';
import { Incident, IncidentSeverity, IncidentState } from '../types/status';
import { fetchActiveIncidents } from '../services/api';

const API_BASE_URL = typeof window !== 'undefined'
  ? ((window as any).__API_BASE_URL__ ?? '')
  : (process.env.API_BASE_URL || '');

interface AuthState {
  loading: boolean;
  authenticated: boolean;
  isAdmin: boolean;
  email?: string;
  name?: string;
  roles?: string[];
  providerName?: string;
  passwordAuthEnabled?: boolean;
}

export const AdminPage: React.FC = () => {
  const [auth, setAuth] = useState<AuthState>({
    loading: true,
    authenticated: false,
    isAdmin: false,
  });

  const [authTab, setAuthTab] = useState<'login' | 'register' | 'forgot'>('login');

  // Login form state
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loginError, setLoginError] = useState('');
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  // Register form state
  const [regUsername, setRegUsername] = useState('');
  const [regPassword, setRegPassword] = useState('');
  const [regEmail, setRegEmail] = useState('');
  const [regName, setRegName] = useState('');
  const [regError, setRegError] = useState('');
  const [regSuccess, setRegSuccess] = useState('');
  const [isRegistering, setIsRegistering] = useState(false);

  // Forgot Password state
  const [forgotEmail, setForgotEmail] = useState('');
  const [resetToken, setResetToken] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [forgotMsg, setForgotMsg] = useState('');
  const [forgotError, setForgotError] = useState('');
  const [isResetting, setIsResetting] = useState(false);

  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [title, setTitle] = useState('');
  const [selectedProducts, setSelectedProducts] = useState<string[]>([]);
  const [selectedChecks, setSelectedChecks] = useState<string[]>([]);
  const [selectedFeatures, setSelectedFeatures] = useState<string[]>([]);
  const [severity, setSeverity] = useState<IncidentSeverity>('minor');
  const [state, setState] = useState<IncidentState>('investigating');
  const [message, setMessage] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [activeUpdateId, setActiveUpdateId] = useState<string | null>(null);
  const [updateMessage, setUpdateMessage] = useState('');
  const [updateState, setUpdateState] = useState<IncidentState>('monitoring');

  // Check current session from /auth/me
  const checkAuth = useCallback(async () => {
    try {
      const [meRes, cfgRes] = await Promise.all([
        fetch(`${API_BASE_URL}/auth/me`, { credentials: 'include' }),
        fetch(`${API_BASE_URL}/auth/config`),
      ]);

      const meData = await meRes.json();
      const cfgData = await cfgRes.json();

      setAuth({
        loading: false,
        authenticated: !!meData.authenticated,
        isAdmin: !!meData.is_admin,
        email: meData.user?.email,
        name: meData.user?.name,
        roles: meData.user?.roles,
        providerName: cfgData.provider_name || 'Enterprise SSO',
        passwordAuthEnabled: cfgData.password_auth_enabled !== false,
      });
    } catch (e) {
      setAuth({ loading: false, authenticated: false, isAdmin: false });
    }
  }, []);

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) return;

    setIsLoggingIn(true);
    setLoginError('');

    try {
      const res = await fetch(`${API_BASE_URL}/auth/login/password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ username, password }),
      });

      if (!res.ok) {
        const err = await res.json();
        setLoginError(err.error || 'Invalid credentials');
        return;
      }

      await checkAuth();
      await loadIncidents();
    } catch (err: any) {
      setLoginError(err.message || 'Connection failed');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!regUsername || !regPassword) return;

    setIsRegistering(true);
    setRegError('');
    setRegSuccess('');

    try {
      const res = await fetch(`${API_BASE_URL}/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          username: regUsername,
          password: regPassword,
          email: regEmail,
          name: regName,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        setRegError(data.error || data.message || 'Registration failed');
        return;
      }

      setRegSuccess('Registration successful! You are now logged in.');
      await checkAuth();
      await loadIncidents();
    } catch (err: any) {
      setRegError(err.message || 'Connection failed');
    } finally {
      setIsRegistering(false);
    }
  };

  const handleForgotPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!forgotEmail) return;

    setIsResetting(true);
    setForgotError('');
    setForgotMsg('');

    try {
      const res = await fetch(`${API_BASE_URL}/auth/forgot-password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: forgotEmail }),
      });

      const data = await res.json();
      if (!res.ok) {
        setForgotError(data.error || 'Request failed');
        return;
      }

      setForgotMsg(`Reset token generated. Enter new password below.`);
      if (data.reset_token) {
        setResetToken(data.reset_token);
      }
    } catch (err: any) {
      setForgotError(err.message || 'Connection failed');
    } finally {
      setIsResetting(false);
    }
  };

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!resetToken || !newPassword) return;

    setIsResetting(true);
    setForgotError('');
    setForgotMsg('');

    try {
      const res = await fetch(`${API_BASE_URL}/auth/reset-password`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ reset_token: resetToken, new_password: newPassword }),
      });

      const data = await res.json();
      if (!res.ok) {
        setForgotError(data.error || 'Password reset failed');
        return;
      }

      setForgotMsg('Password reset successfully! Logging you in...');
      await checkAuth();
      await loadIncidents();
    } catch (err: any) {
      setForgotError(err.message || 'Connection failed');
    } finally {
      setIsResetting(false);
    }
  };

  const loadIncidents = useCallback(async () => {
    try {
      const data = await fetchActiveIncidents('*');
      setIncidents(data);
    } catch (e) {
      console.error('Failed to load incidents:', e);
    }
  }, []);

  useEffect(() => {
    checkAuth();
    loadIncidents();
  }, [checkAuth, loadIncidents]);

  const handleCreateIncident = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title || !message) return;

    setIsSubmitting(true);
    try {
      const newInc: Incident = {
        id: `inc-${Date.now()}`,
        tenant: '*',
        product: selectedProducts.length === 1 ? selectedProducts[0] : undefined,
        products: selectedProducts.length > 0 ? selectedProducts : undefined,
        check_ids: selectedChecks.length > 0 ? selectedChecks : undefined,
        features: selectedFeatures.length > 0 ? selectedFeatures : undefined,
        title,
        severity,
        state,
        message,
        active: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        updates: [
          {
            timestamp: new Date().toISOString(),
            state,
            message,
          },
        ],
      };

      const res = await fetch(`${API_BASE_URL}/api/incidents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(newInc),
      });

      if (!res.ok) {
        const err = await res.json();
        alert(`Failed to post incident: ${err.error || res.statusText}`);
        return;
      }

      // Reset form
      setTitle('');
      setMessage('');
      setSelectedProducts([]);
      setSelectedChecks([]);
      setSelectedFeatures([]);
      await loadIncidents();
    } catch (err: any) {
      alert(`Error publishing incident: ${err.message}`);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleAddUpdate = async (incident: Incident) => {
    if (!updateMessage) return;

    const updatedInc: Incident = {
      ...incident,
      state: updateState,
      message: updateMessage,
      updated_at: new Date().toISOString(),
      updates: [
        ...(incident.updates || []),
        {
          timestamp: new Date().toISOString(),
          state: updateState,
          message: updateMessage,
        },
      ],
    };

    try {
      const res = await fetch(`${API_BASE_URL}/api/incidents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(updatedInc),
      });

      if (res.ok) {
        setActiveUpdateId(null);
        setUpdateMessage('');
        await loadIncidents();
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleResolveIncident = async (id: string) => {
    if (!confirm('Resolve and remove this announcement banner?')) return;

    try {
      await fetch(`${API_BASE_URL}/api/incidents/${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      await loadIncidents();
    } catch (err) {
      console.error(err);
    }
  };

  const handleLogout = async () => {
    await fetch(`${API_BASE_URL}/auth/logout`, {
      method: 'POST',
      credentials: 'include',
    });
    setAuth({ loading: false, authenticated: false, isAdmin: false });
  };

  if (auth.loading) {
    return (
      <div className="status-container" style={{ textAlign: 'center', paddingTop: '80px' }}>
        <p style={{ color: 'var(--text-muted)' }}>Checking authentication credentials...</p>
      </div>
    );
  }

  // Login view if unauthenticated
  if (!auth.authenticated || !auth.isAdmin) {
    return (
      <div className="status-container">
        <header className="app-header">
          <div className="header-inner">
            <Link to="/" className="brand-section">
              <div className="brand-logo-badge">
                <svg width="22" height="22" viewBox="0 0 24 24" style={{ width: '22px', height: '22px', display: 'block' }}>
                  <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="#ffffff" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </div>
              <div className="brand-name">Platform Status Admin</div>
            </Link>
            <Link to="/" className="selector-pill" style={{ textDecoration: 'none' }}>
              ← Back to Public Status
            </Link>
          </div>
        </header>

        <div className="admin-login-card">
          <div className="login-shield-icon">🔐</div>
          <h2>Internal Incident Management</h2>
          <p>
            Authenticate with enterprise SSO or credentials to control platform announcement banners.
          </p>

          {/* Navigation Tabs */}
          <div style={{ display: 'flex', gap: '8px', marginBottom: '24px', justifyContent: 'center' }}>
            <button
              className={`filter-chip ${authTab === 'login' ? 'active' : ''}`}
              onClick={() => { setAuthTab('login'); setLoginError(''); }}
            >
              Sign In
            </button>
            <button
              className={`filter-chip ${authTab === 'register' ? 'active' : ''}`}
              onClick={() => { setAuthTab('register'); setRegError(''); setRegSuccess(''); }}
            >
              Register Account
            </button>
            <button
              className={`filter-chip ${authTab === 'forgot' ? 'active' : ''}`}
              onClick={() => { setAuthTab('forgot'); setForgotError(''); setForgotMsg(''); }}
            >
              Forgot Password
            </button>
          </div>

          {/* TAB 1: Sign In */}
          {authTab === 'login' && (
            <div>
              <a
                href={`${API_BASE_URL}/auth/login/sso`}
                className="sso-login-btn"
                style={{ width: '100%' }}
              >
                <span>Sign in with {auth.providerName}</span>
              </a>

              {auth.passwordAuthEnabled && (
                <div style={{ marginTop: '28px' }}>
                  <div className="auth-divider">
                    <span>OR SIGN IN WITH USERNAME / PASSWORD</span>
                  </div>

                  {loginError && (
                    <div style={{ padding: '10px', backgroundColor: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.4)', borderRadius: '8px', color: 'var(--color-outage)', fontSize: '0.85rem', marginBottom: '16px' }}>
                      ⚠ {loginError}
                    </div>
                  )}

                  <form onSubmit={handlePasswordLogin} style={{ display: 'grid', gap: '14px', textAlign: 'left' }}>
                    <div>
                      <label className="admin-label">Username</label>
                      <input
                        type="text"
                        className="admin-input"
                        placeholder="Username (e.g. admin)"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        required
                      />
                    </div>

                    <div>
                      <label className="admin-label">Password</label>
                      <input
                        type="password"
                        className="admin-input"
                        placeholder="Password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                      />
                    </div>

                    <button
                      type="submit"
                      className="selector-pill"
                      style={{ width: '100%', padding: '12px', fontWeight: 700, backgroundColor: 'rgba(255, 255, 255, 0.1)', borderColor: 'var(--border-hover)' }}
                      disabled={isLoggingIn}
                    >
                      {isLoggingIn ? 'Verifying...' : 'Sign In with Credentials'}
                    </button>
                  </form>
                </div>
              )}
            </div>
          )}

          {/* TAB 2: Register */}
          {authTab === 'register' && (
            <div style={{ textAlign: 'left' }}>
              {regError && (
                <div style={{ padding: '10px', backgroundColor: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.4)', borderRadius: '8px', color: 'var(--color-outage)', fontSize: '0.85rem', marginBottom: '16px' }}>
                  ⚠ {regError}
                </div>
              )}
              {regSuccess && (
                <div style={{ padding: '10px', backgroundColor: 'rgba(16, 185, 129, 0.15)', border: '1px solid rgba(16, 185, 129, 0.4)', borderRadius: '8px', color: 'var(--color-operational)', fontSize: '0.85rem', marginBottom: '16px' }}>
                  ✓ {regSuccess}
                </div>
              )}

              <form onSubmit={handleRegister} style={{ display: 'grid', gap: '12px' }}>
                <div>
                  <label className="admin-label">Username</label>
                  <input
                    type="text"
                    className="admin-input"
                    placeholder="New username"
                    value={regUsername}
                    onChange={(e) => setRegUsername(e.target.value)}
                    required
                  />
                </div>

                <div>
                  <label className="admin-label">Email</label>
                  <input
                    type="email"
                    className="admin-input"
                    placeholder="admin@company.com"
                    value={regEmail}
                    onChange={(e) => setRegEmail(e.target.value)}
                    required
                  />
                </div>

                <div>
                  <label className="admin-label">Display Name</label>
                  <input
                    type="text"
                    className="admin-input"
                    placeholder="Full Name"
                    value={regName}
                    onChange={(e) => setRegName(e.target.value)}
                  />
                </div>

                <div>
                  <label className="admin-label">Password</label>
                  <input
                    type="password"
                    className="admin-input"
                    placeholder="Password (minimum 6 characters)"
                    value={regPassword}
                    onChange={(e) => setRegPassword(e.target.value)}
                    required
                  />
                </div>

                <button
                  type="submit"
                  className="sso-login-btn"
                  style={{ width: '100%', marginTop: '6px' }}
                  disabled={isRegistering}
                >
                  {isRegistering ? 'Registering...' : 'Register Operator Account'}
                </button>
              </form>
            </div>
          )}

          {/* TAB 3: Forgot Password */}
          {authTab === 'forgot' && (
            <div style={{ textAlign: 'left' }}>
              {forgotError && (
                <div style={{ padding: '10px', backgroundColor: 'rgba(239, 68, 68, 0.15)', border: '1px solid rgba(239, 68, 68, 0.4)', borderRadius: '8px', color: 'var(--color-outage)', fontSize: '0.85rem', marginBottom: '16px' }}>
                  ⚠ {forgotError}
                </div>
              )}
              {forgotMsg && (
                <div style={{ padding: '10px', backgroundColor: 'rgba(59, 130, 246, 0.15)', border: '1px solid rgba(59, 130, 246, 0.4)', borderRadius: '8px', color: '#60a5fa', fontSize: '0.85rem', marginBottom: '16px' }}>
                  ℹ {forgotMsg}
                </div>
              )}

              {!resetToken ? (
                <form onSubmit={handleForgotPassword} style={{ display: 'grid', gap: '14px' }}>
                  <div>
                    <label className="admin-label">Account Email Address</label>
                    <input
                      type="email"
                      className="admin-input"
                      placeholder="Enter registered email"
                      value={forgotEmail}
                      onChange={(e) => setForgotEmail(e.target.value)}
                      required
                    />
                  </div>

                  <button
                    type="submit"
                    className="sso-login-btn"
                    style={{ width: '100%' }}
                    disabled={isResetting}
                  >
                    {isResetting ? 'Processing...' : 'Request Password Reset'}
                  </button>
                </form>
              ) : (
                <form onSubmit={handleResetPassword} style={{ display: 'grid', gap: '14px' }}>
                  <div>
                    <label className="admin-label">Reset Token</label>
                    <input
                      type="text"
                      className="admin-input"
                      value={resetToken}
                      onChange={(e) => setResetToken(e.target.value)}
                      required
                    />
                  </div>

                  <div>
                    <label className="admin-label">New Password</label>
                    <input
                      type="password"
                      className="admin-input"
                      placeholder="Enter new password"
                      value={newPassword}
                      onChange={(e) => setNewPassword(e.target.value)}
                      required
                    />
                  </div>

                  <button
                    type="submit"
                    className="sso-login-btn"
                    style={{ width: '100%' }}
                    disabled={isResetting}
                  >
                    {isResetting ? 'Saving...' : 'Set New Password & Login'}
                  </button>
                </form>
              )}
            </div>
          )}

          <div style={{ marginTop: '28px', fontSize: '0.75rem', color: 'var(--text-dim)' }}>
            gRPC Stateless Microservice &bull; Enterprise OIDC &bull; Local Credentials
          </div>
        </div>
      </div>
    );
  }

  // Admin incident management console
  return (
    <div>
      <header className="app-header">
        <div className="header-inner">
          <Link to="/" className="brand-section">
            <div className="brand-logo-badge">
              <svg width="22" height="22" viewBox="0 0 24 24" style={{ width: '22px', height: '22px', display: 'block' }}>
                <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="#ffffff" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
              </svg>
            </div>
            <div>
              <div className="brand-name">
                Incident Control Center
                <span className="brand-tenant-tag" style={{ color: 'var(--color-operational)' }}>Admin</span>
              </div>
            </div>
          </Link>

          <div className="header-controls">
            <span style={{ fontSize: '0.82rem', color: 'var(--text-muted)' }}>
              {auth.email}
            </span>
            <Link to="/" className="selector-pill" style={{ textDecoration: 'none' }}>
              Public View ↗
            </Link>
            <button className="selector-pill" onClick={handleLogout}>
              Sign Out
            </button>
          </div>
        </div>
      </header>

      <main className="status-container">
        {/* Create Incident Form */}
        <div className="admin-card">
          <h2 style={{ fontSize: '1.25rem', fontWeight: 700, marginBottom: '6px' }}>
            Publish Incident Announcement Banner
          </h2>
          <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', marginBottom: '20px' }}>
            Broadcasting an incident immediately surfaces a prominent banner across the top of public status dashboards.
          </p>

          <form onSubmit={handleCreateIncident} style={{ display: 'grid', gap: '16px' }}>
            <div>
              <label className="admin-label">Incident Title</label>
              <input
                type="text"
                className="admin-input"
                placeholder="e.g. Elevated Latency on Credit Card Checkout"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
              />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
              <div>
                <label className="admin-label">Severity Level</label>
                <select
                  className="admin-input"
                  value={severity}
                  onChange={(e) => setSeverity(e.target.value as IncidentSeverity)}
                >
                  <option value="critical">Critical (Major Outage)</option>
                  <option value="major">Major (Severe Degradation)</option>
                  <option value="minor">Minor (Elevated Latency)</option>
                  <option value="maintenance">Maintenance (Scheduled Window)</option>
                  <option value="info">Info (Informational Notice)</option>
                </select>
              </div>

              <div>
                <label className="admin-label">Investigation State</label>
                <select
                  className="admin-input"
                  value={state}
                  onChange={(e) => setState(e.target.value as IncidentState)}
                >
                  <option value="investigating">Investigating</option>
                  <option value="identified">Identified</option>
                  <option value="monitoring">Monitoring</option>
                  <option value="resolved">Resolved</option>
                </select>
              </div>
            </div>

            {/* Linked Products Selector */}
            <div>
              <label className="admin-label">Link to Affected Products (Multi-select)</label>
              <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginTop: '6px' }}>
                {['cloud-api', 'auth-service', 'billing-engine'].map((p) => {
                  const isSelected = selectedProducts.includes(p);
                  return (
                    <button
                      key={p}
                      type="button"
                      className={`filter-chip ${isSelected ? 'active' : ''}`}
                      onClick={() => {
                        setSelectedProducts((prev) =>
                          isSelected ? prev.filter((item) => item !== p) : [...prev, p]
                        );
                      }}
                    >
                      {isSelected ? '✓ ' : '+ '} {p}
                    </button>
                  );
                })}
                {selectedProducts.length === 0 && (
                  <span style={{ fontSize: '0.75rem', color: 'var(--text-dim)', alignSelf: 'center' }}>
                    (No product selected = Global Platform-wide banner)
                  </span>
                )}
              </div>
            </div>

            {/* Linked Health Checks Selector */}
            <div>
              <label className="admin-label">Link to Specific Health Checks (Multi-select)</label>
              <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginTop: '6px' }}>
                {[
                  'gateway-routing-check',
                  'redis-rate-limiter-check',
                  'oidc-jwks-check',
                  'session-cache-check',
                  'stripe-webhook-check',
                  'tax-jurisdiction-check-us',
                  'tax-jurisdiction-check-eu',
                ].map((cid) => {
                  const isSelected = selectedChecks.includes(cid);
                  return (
                    <button
                      key={cid}
                      type="button"
                      className={`filter-chip ${isSelected ? 'active' : ''}`}
                      style={{ borderColor: isSelected ? '#60a5fa' : undefined, color: isSelected ? '#93c5fd' : undefined }}
                      onClick={() => {
                        setSelectedChecks((prev) =>
                          isSelected ? prev.filter((item) => item !== cid) : [...prev, cid]
                        );
                      }}
                    >
                      {isSelected ? '✓ ' : '🔍 '} {cid}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Linked Features Selector */}
            <div>
              <label className="admin-label">Link to Features (Multi-select)</label>
              <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginTop: '6px' }}>
                {['api-gateway', 'rate-limiter', 'oauth2', 'sessions', 'stripe-connector', 'tax-calc'].map((fid) => {
                  const isSelected = selectedFeatures.includes(fid);
                  return (
                    <button
                      key={fid}
                      type="button"
                      className={`filter-chip ${isSelected ? 'active' : ''}`}
                      style={{ borderColor: isSelected ? '#c084fc' : undefined, color: isSelected ? '#e9d5ff' : undefined }}
                      onClick={() => {
                        setSelectedFeatures((prev) =>
                          isSelected ? prev.filter((item) => item !== fid) : [...prev, fid]
                        );
                      }}
                    >
                      {isSelected ? '✓ ' : '⚡ '} {fid}
                    </button>
                  );
                })}
              </div>
            </div>

            <div>
              <label className="admin-label">Initial Announcement Message</label>
              <textarea
                className="admin-input"
                rows={3}
                placeholder="Describe current symptoms, engineering findings, and next update expectation..."
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                required
              />
            </div>

            <button
              type="submit"
              className="sso-login-btn"
              style={{ width: 'fit-content', padding: '10px 24px' }}
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Publishing...' : 'Publish Announcement Banner'}
            </button>
          </form>
        </div>

        {/* Active Incidents List */}
        <div style={{ marginTop: '36px' }}>
          <h2 style={{ fontSize: '1.25rem', fontWeight: 700, marginBottom: '16px' }}>
            Active Incident Banners ({incidents.filter((i) => i.active).length})
          </h2>

          {incidents.filter((i) => i.active).length === 0 ? (
            <div className="empty-state">
              <h3>No Active Incidents</h3>
              <p>All platform systems are green and zero operator banners are currently broadcast.</p>
            </div>
          ) : (
            <div style={{ display: 'grid', gap: '20px' }}>
              {incidents
                .filter((i) => i.active)
                .map((inc) => (
                  <div key={inc.id} className="admin-card">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '12px' }}>
                      <div>
                        <div style={{ display: 'flex', gap: '8px', alignItems: 'center', marginBottom: '8px', flexWrap: 'wrap' }}>
                          <span className={`announcement-severity-pill severity-${inc.severity}`}>
                            {inc.severity.toUpperCase()}
                          </span>
                          <span className={`announcement-state-pill state-${inc.state}`}>
                            {inc.state.toUpperCase()}
                          </span>

                          {/* Linked Products */}
                          {inc.products && inc.products.length > 0 ? (
                            inc.products.map((p) => (
                              <span key={p} className="announcement-product-tag">📦 {p}</span>
                            ))
                          ) : inc.product ? (
                            <span className="announcement-product-tag">📦 {inc.product}</span>
                          ) : (
                            <span className="announcement-product-tag">🌐 Global</span>
                          )}

                          {/* Linked Checks */}
                          {inc.check_ids && inc.check_ids.map((cid) => (
                            <span key={cid} className="announcement-product-tag" style={{ borderColor: 'rgba(59, 130, 246, 0.4)', color: '#93c5fd' }}>
                              🔍 {cid}
                            </span>
                          ))}

                          {/* Linked Features */}
                          {inc.features && inc.features.map((fid) => (
                            <span key={fid} className="announcement-product-tag" style={{ borderColor: 'rgba(168, 85, 247, 0.4)', color: '#d8b4fe' }}>
                              ⚡ {fid}
                            </span>
                          ))}
                        </div>
                        <h3 style={{ fontSize: '1.1rem', fontWeight: 700 }}>{inc.title}</h3>
                        <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem', marginTop: '4px' }}>{inc.message}</p>
                      </div>

                      <div style={{ display: 'flex', gap: '8px' }}>
                        <button
                          className="filter-chip"
                          onClick={() => setActiveUpdateId(activeUpdateId === inc.id ? null : inc.id)}
                        >
                          {activeUpdateId === inc.id ? 'Cancel Update' : '+ Post Update'}
                        </button>
                        <button
                          className="filter-chip"
                          style={{ borderColor: 'rgba(239, 68, 68, 0.4)', color: 'var(--color-outage)' }}
                          onClick={() => handleResolveIncident(inc.id)}
                        >
                          Resolve & Remove
                        </button>
                      </div>
                    </div>

                    {/* Progress Update Form */}
                    {activeUpdateId === inc.id && (
                      <div style={{ marginTop: '16px', padding: '16px', backgroundColor: 'rgba(0,0,0,0.3)', borderRadius: '8px' }}>
                        <label className="admin-label">New Investigation State</label>
                        <select
                          className="admin-input"
                          style={{ marginBottom: '10px' }}
                          value={updateState}
                          onChange={(e) => setUpdateState(e.target.value as IncidentState)}
                        >
                          <option value="investigating">Investigating</option>
                          <option value="identified">Identified</option>
                          <option value="monitoring">Monitoring</option>
                          <option value="resolved">Resolved</option>
                        </select>

                        <label className="admin-label">Progress Update Message</label>
                        <textarea
                          className="admin-input"
                          rows={2}
                          placeholder="e.g. Traffic rerouted to backup providers. Queue backlog is recovering."
                          value={updateMessage}
                          onChange={(e) => setUpdateMessage(e.target.value)}
                        />

                        <button
                          className="sso-login-btn"
                          style={{ marginTop: '10px', fontSize: '0.8rem', padding: '6px 16px' }}
                          onClick={() => handleAddUpdate(inc)}
                        >
                          Commit Progress Update
                        </button>
                      </div>
                    )}

                    {/* Timeline History */}
                    {inc.updates && inc.updates.length > 0 && (
                      <div style={{ marginTop: '16px', borderTop: '1px solid var(--border-subtle)', paddingTop: '12px' }}>
                        <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-dim)', marginBottom: '8px' }}>
                          Timeline History:
                        </div>
                        <div style={{ display: 'grid', gap: '6px' }}>
                          {inc.updates.map((u, idx) => (
                            <div key={idx} style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                              <span style={{ color: 'var(--text-dim)', fontFamily: 'var(--font-mono)', marginRight: '8px' }}>
                                {new Date(u.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                              </span>
                              <strong>[{u.state.toUpperCase()}]</strong> {u.message}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
            </div>
          )}
        </div>
      </main>
    </div>
  );
};
