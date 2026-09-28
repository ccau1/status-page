import React from 'react';
import { useNavigate } from 'react-router';

interface HeaderProps {
  tenant: string;
  locale: string;
  product?: string;
  onRefresh: () => void;
  isRefreshing: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  tenant,
  locale,
  product,
  onRefresh,
  isRefreshing,
}) => {
  const navigate = useNavigate();

  const handleLocaleChange = (newLocale: string) => {
    const prodPart = product ? `/${product}` : '';
    navigate(`/${tenant}/${newLocale}/status${prodPart}`);
  };

  return (
    <header className="app-header">
      <div className="header-inner">
        <div className="brand-section">
          <div className="brand-logo-badge">
            <svg width="22" height="22" viewBox="0 0 24 24" style={{ width: '22px', height: '22px', display: 'block' }}>
              <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="#ffffff" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </div>
          <div>
            <div className="brand-name">
              Platform Status
            </div>
          </div>
        </div>

        <div className="header-controls">
          {/* Locale Picker */}
          <select
            className="selector-pill"
            value={locale}
            onChange={(e) => handleLocaleChange(e.target.value)}
            aria-label="Select Locale"
          >
            <option value="en">English (EN)</option>
            <option value="es">Español (ES)</option>
            <option value="fr">Français (FR)</option>
            <option value="de">Deutsch (DE)</option>
            <option value="ja">日本語 (JA)</option>
          </select>

          {/* Refresh button */}
          <button
            className="selector-pill"
            onClick={onRefresh}
            title="Refresh status data"
            style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
          >
            <span style={{ display: 'inline-block', transform: isRefreshing ? 'rotate(360deg)' : 'none', transition: 'transform 0.6s ease' }}>
              ↻
            </span>
            {isRefreshing ? 'Checking...' : 'Refresh'}
          </button>

          <div className="live-indicator">
            <span className="pulse-dot"></span>
            LIVE
          </div>
        </div>
      </div>
    </header>
  );
};
