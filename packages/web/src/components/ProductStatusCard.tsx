import React, { useState } from 'react';
import { Status, FeatureStatus, Incident } from '../types/status';
import { UptimeBar } from './UptimeBar';
import { FeatureItem } from './FeatureItem';

interface ProductStatusCardProps {
  status: Status;
  filteredFeatures: FeatureStatus[];
  linkedIncidents?: Incident[];
}

export const ProductStatusCard: React.FC<ProductStatusCardProps> = ({
  status,
  filteredFeatures,
  linkedIncidents = [],
}) => {
  const [showChecks, setShowChecks] = useState(false);

  return (
    <div className="product-card">
      {/* Alert banner if an active incident explicitly links to this product */}
      {linkedIncidents.length > 0 && (
        <div style={{ padding: '8px 24px', backgroundColor: 'rgba(245, 158, 11, 0.15)', borderBottom: '1px solid rgba(245, 158, 11, 0.3)', display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.8rem', color: '#fbbf24', fontWeight: 600 }}>
          <span>⚠ Linked to Active Incident:</span>
          <span>{linkedIncidents.map(i => i.title).join(' | ')}</span>
        </div>
      )}

      <div className="product-card-header">
        <div className="product-info">
          <h2>{formatProductName(status.product)}</h2>
          <p>{status.message || 'Product service nodes monitored continuously'}</p>
        </div>

        <div className="product-header-actions">
          <div className={`state-badge state-${status.current_state}`} style={{ fontSize: '0.85rem', padding: '6px 14px' }}>
            <span className="pulse-dot"></span>
            {status.current_state}
          </div>

          {status.check_results && status.check_results.length > 0 && (
            <button
              onClick={() => setShowChecks(!showChecks)}
              className="filter-chip"
              style={{ fontSize: '0.75rem' }}
              title="Inspect raw check definitions"
            >
              {showChecks ? 'Hide Checks' : `Checks (${status.check_results.length})`}
            </button>
          )}
        </div>
      </div>

      {/* 60-day visual uptime bar */}
      <UptimeBar currentState={status.current_state} days={60} />

      {/* Filtered Features List */}
      <div className="features-container">
        {filteredFeatures.length === 0 ? (
          <div style={{ padding: '24px', textAlign: 'center', color: 'var(--text-dim)', fontSize: '0.85rem' }}>
            No features in this product match the current feature filter.
          </div>
        ) : (
          filteredFeatures.map((feat) => (
            <FeatureItem key={feat.id} feature={feat} />
          ))
        )}
      </div>

      {/* Detailed Check Drawer */}
      {showChecks && status.check_results && (
        <div style={{ padding: '16px 24px', backgroundColor: 'rgba(0,0,0,0.3)', borderTop: '1px solid var(--border-subtle)', fontSize: '0.8rem' }}>
          <div style={{ fontWeight: 600, color: 'var(--text-muted)', marginBottom: '10px' }}>
            Modular Health Checks Execution Trace:
          </div>
          <div style={{ display: 'grid', gap: '8px' }}>
            {status.check_results.map((c) => (
              <div
                key={c.check_id}
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  padding: '8px 12px',
                  borderRadius: '6px',
                  backgroundColor: 'var(--bg-card)',
                  border: '1px solid var(--border-subtle)',
                }}
              >
                <div>
                  <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, color: 'var(--text-main)', marginRight: '8px' }}>
                    [{c.type.toUpperCase()}] {c.check_id}
                  </span>
                  <span style={{ color: 'var(--text-dim)' }}>&bull; Feature: {c.feature}</span>
                  {c.region && (
                    <span style={{ color: 'var(--text-dim)' }}> &bull; Region: {c.region}</span>
                  )}
                </div>
                <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                  <span style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}>{c.latency_ms}ms</span>
                  <span style={{ color: c.success ? 'var(--color-operational)' : 'var(--color-outage)', fontWeight: 600 }}>
                    {c.message || (c.success ? 'PASSED' : 'FAILED')}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

function formatProductName(product: string): string {
  return product
    .split(/[-_]/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}
