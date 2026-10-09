import React from 'react';
import { useNavigate } from 'react-router';
import { StatusState } from '../types/status';

interface FilterToolbarProps {
  tenant: string;
  locale: string;
  currentProduct?: string;
  allProducts: string[];
  searchQuery: string;
  onSearchChange: (q: string) => void;
  selectedState: StatusState | 'all';
  onStateSelect: (s: StatusState | 'all') => void;
  featureCounts: {
    all: number;
    operational: number;
    degraded: number;
    outage: number;
  };
  allRegions: string[];
  selectedRegion: string | 'all';
  onRegionSelect: (r: string | 'all') => void;
}

export const FilterToolbar: React.FC<FilterToolbarProps> = ({
  tenant,
  locale,
  currentProduct,
  allProducts,
  searchQuery,
  onSearchChange,
  selectedState,
  onStateSelect,
  featureCounts,
  allRegions,
  selectedRegion,
  onRegionSelect,
}) => {
  const navigate = useNavigate();

  const handleProductSelect = (productSlug?: string) => {
    if (!productSlug) {
      navigate('/status');
    } else {
      navigate(`/status/${productSlug}`);
    }
  };

  return (
    <div className="filter-toolbar">
      <div className="search-and-products">
        {/* In-page Feature Search */}
        <div className="search-input-box">
          <svg width="18" height="18" viewBox="0 0 24 24" style={{ width: '18px', height: '18px' }}>
            <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" fill="none"/>
          </svg>
          <input
            type="text"
            className="search-input"
            placeholder="Filter features by name or keyword..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            aria-label="Filter features"
          />
          {searchQuery && (
            <button
              onClick={() => onSearchChange('')}
              style={{
                position: 'absolute',
                right: '12px',
                top: '50%',
                transform: 'translateY(-50%)',
                background: 'none',
                border: 'none',
                color: 'var(--text-dim)',
                cursor: 'pointer',
                fontSize: '1rem',
              }}
              title="Clear search"
            >
              ✕
            </button>
          )}
        </div>

        {/* Product Switcher Tabs */}
        <div className="product-tabs" role="tablist" aria-label="Filter by Product">
          <button
            role="tab"
            aria-selected={!currentProduct}
            className={`tab-btn ${!currentProduct ? 'active' : ''}`}
            onClick={() => handleProductSelect(undefined)}
          >
            All Products ({allProducts.length})
          </button>
          {allProducts.map((prod) => (
            <button
              key={prod}
              role="tab"
              aria-selected={currentProduct === prod}
              className={`tab-btn ${currentProduct === prod ? 'active' : ''}`}
              onClick={() => handleProductSelect(prod)}
            >
              {formatTitle(prod)}
            </button>
          ))}
        </div>
      </div>

      {/* Region Filter Chips (only when per-region check data exists) */}
      {allRegions.length > 0 && (
        <div className="feature-state-filters" role="group" aria-label="Filter by Region">
          <span style={{ fontSize: '0.78rem', color: 'var(--text-dim)', marginRight: '6px', fontWeight: 600 }}>
            Region Filter:
          </span>

          <button
            className={`filter-chip ${selectedRegion === 'all' ? 'active' : ''}`}
            onClick={() => onRegionSelect('all')}
          >
            All Regions
          </button>

          {allRegions.map((region) => (
            <button
              key={region}
              className={`filter-chip ${selectedRegion === region ? 'active' : ''}`}
              onClick={() => onRegionSelect(region)}
            >
              {region}
            </button>
          ))}
        </div>
      )}

      {/* Feature Health State Filter Chips */}
      <div className="feature-state-filters">
        <span style={{ fontSize: '0.78rem', color: 'var(--text-dim)', marginRight: '6px', fontWeight: 600 }}>
          Feature Status Filter:
        </span>

        <button
          className={`filter-chip ${selectedState === 'all' ? 'active' : ''}`}
          onClick={() => onStateSelect('all')}
        >
          All Features <span className="filter-count">{featureCounts.all}</span>
        </button>

        <button
          className={`filter-chip ${selectedState === 'operational' ? 'active' : ''}`}
          onClick={() => onStateSelect('operational')}
          style={{ color: selectedState === 'operational' ? 'var(--color-operational)' : undefined }}
        >
          <span className="pulse-dot" style={{ width: '6px', height: '6px', color: 'var(--color-operational)' }}></span>
          Operational <span className="filter-count">{featureCounts.operational}</span>
        </button>

        <button
          className={`filter-chip ${selectedState === 'degraded' ? 'active' : ''}`}
          onClick={() => onStateSelect('degraded')}
          style={{ color: selectedState === 'degraded' ? 'var(--color-degraded)' : undefined }}
        >
          <span className="pulse-dot" style={{ width: '6px', height: '6px', color: 'var(--color-degraded)' }}></span>
          Degraded <span className="filter-count">{featureCounts.degraded}</span>
        </button>

        <button
          className={`filter-chip ${selectedState === 'outage' ? 'active' : ''}`}
          onClick={() => onStateSelect('outage')}
          style={{ color: selectedState === 'outage' ? 'var(--color-outage)' : undefined }}
        >
          <span className="pulse-dot" style={{ width: '6px', height: '6px', color: 'var(--color-outage)' }}></span>
          Outages <span className="filter-count">{featureCounts.outage}</span>
        </button>
      </div>
    </div>
  );
};

function formatTitle(s: string): string {
  return s
    .split(/[-_]/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}
