import React from 'react';
import { Link } from 'react-router';

export const Footer: React.FC = () => {
  return (
    <footer className="app-footer">
      <div>
        <strong>Cloud Platform Status Engine</strong> &bull; Real-time status powered by Go Hexagonal Core & Worker
      </div>

      <div className="footer-links">
        <Link to="/admin" style={{ color: 'var(--text-muted)' }}>
          Admin Portal
        </Link>
        <a href="#incident-history" onClick={(e) => { e.preventDefault(); alert('Active incident announcements are broadcast at the top of this status dashboard.'); }}>
          Incident Log
        </a>
        <a href="#api" onClick={(e) => { e.preventDefault(); alert('API endpoints available at /api/status'); }}>
          Status API
        </a>
        <a href="#rss" onClick={(e) => { e.preventDefault(); alert('Subscribe to RSS / Webhook updates'); }}>
          Subscribe
        </a>
      </div>
    </footer>
  );
};
