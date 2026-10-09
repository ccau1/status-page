import React from 'react';
import { FeatureStatus } from '../types/status';

interface FeatureItemProps {
  feature: FeatureStatus;
}

export const FeatureItem: React.FC<FeatureItemProps> = ({ feature }) => {
  return (
    <div className="feature-row">
      <div className="feature-meta">
        <div className="feature-title-line">
          <h3>{feature.name || feature.id}</h3>
          {feature.region && (
            <span className="latency-pill" style={{ marginLeft: '8px' }} title="Region-specific status">
              {feature.region}
            </span>
          )}
        </div>
        {feature.description && (
          <div className="feature-description">{feature.description}</div>
        )}
        {feature.message && feature.state !== 'operational' && (
          <div style={{ fontSize: '0.78rem', color: 'var(--color-degraded)', marginTop: '4px' }}>
            ⚠ {feature.message}
          </div>
        )}
      </div>

      <div className="feature-status-side">
        {feature.latency_ms !== undefined && feature.latency_ms > 0 && (
          <span className="latency-pill">{feature.latency_ms}ms</span>
        )}
        <div className={`state-badge state-${feature.state}`}>
          <span className="pulse-dot" style={{ width: '6px', height: '6px' }}></span>
          {feature.state}
        </div>
      </div>
    </div>
  );
};
