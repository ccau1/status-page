import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useParams } from 'react-router';
import { Status, StatusState, FeatureStatus } from '../types/status';
import { fetchStatusesByTenant, fetchActiveIncidents, FALLBACK_STATUSES, FALLBACK_INCIDENTS } from '../services/api';
import { Header } from '../components/Header';
import { AnnouncementBanner } from '../components/AnnouncementBanner';
import { SystemBanner } from '../components/SystemBanner';
import { MetricsOverview } from '../components/MetricsOverview';
import { FilterToolbar } from '../components/FilterToolbar';
import { ProductStatusCard } from '../components/ProductStatusCard';
import { Footer } from '../components/Footer';

export const StatusPage: React.FC = () => {
  const params = useParams<{
    tenant?: string;
    locale?: string;
    product?: string;
  }>();

  const tenant = params.tenant || 'default';
  const locale = params.locale || 'en';
  const product = params.product;

  const [statuses, setStatuses] = useState<Status[]>(() =>
    FALLBACK_STATUSES.map((s) => ({ ...s, tenant }))
  );
  const [incidents, setIncidents] = useState(() => FALLBACK_INCIDENTS);
  const [loading, setLoading] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // In-page feature filters
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedState, setSelectedState] = useState<StatusState | 'all'>('all');

  const loadData = useCallback(async (isSilent = false) => {
    if (!isSilent) setLoading(true);
    setIsRefreshing(true);
    try {
      const [statusData, incidentData] = await Promise.all([
        fetchStatusesByTenant(tenant),
        fetchActiveIncidents(tenant),
      ]);
      setStatuses(statusData);
      setIncidents(incidentData);
    } catch (e) {
      console.error('Failed to load status/incident data:', e);
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  }, [tenant]);

  useEffect(() => {
    loadData();
    const timer = setInterval(() => loadData(true), 30000);
    return () => clearInterval(timer);
  }, [loadData]);

  // List of all available product slugs for this tenant
  const allProducts = useMemo(() => {
    return statuses.map((s) => s.product);
  }, [statuses]);

  // Current products to display (if URL specified product, filter to that one)
  const displayedStatuses = useMemo(() => {
    if (!product) return statuses;
    return statuses.filter((s) => s.product.toLowerCase() === product.toLowerCase());
  }, [statuses, product]);

  // Calculate overall state across displayed products
  const overallState = useMemo<StatusState>(() => {
    if (displayedStatuses.length === 0) return 'operational';
    const states = displayedStatuses.map((s) => s.current_state);
    if (states.includes('outage')) return 'outage';
    if (states.includes('degraded')) return 'degraded';
    if (states.includes('maintenance')) return 'maintenance';
    return 'operational';
  }, [displayedStatuses]);

  // Compute affected features for pre-made banner message when checks are down
  const affectedFeatures = useMemo(() => {
    const list: string[] = [];
    displayedStatuses.forEach((st) => {
      Object.values(st.features).forEach((feat) => {
        if (feat.state !== 'operational') {
          list.push(`${feat.name || feat.id} (${st.product})`);
        }
      });
    });
    return list;
  }, [displayedStatuses]);

  // Compute in-page feature counts across current displayed products
  const featureCounts = useMemo(() => {
    let all = 0;
    let operational = 0;
    let degraded = 0;
    let outage = 0;

    displayedStatuses.forEach((prod) => {
      Object.values(prod.features).forEach((feat) => {
        all++;
        if (feat.state === 'operational') operational++;
        else if (feat.state === 'degraded') degraded++;
        else if (feat.state === 'outage') outage++;
      });
    });

    return { all, operational, degraded, outage };
  }, [displayedStatuses]);

  // Helper to filter features within a product card
  const getFilteredFeatures = useCallback((featuresMap: Record<string, FeatureStatus>): FeatureStatus[] => {
    const query = searchQuery.trim().toLowerCase();

    return Object.values(featuresMap).filter((feat) => {
      // 1. Filter by search query
      if (query) {
        const nameMatch = feat.name?.toLowerCase().includes(query);
        const idMatch = feat.id.toLowerCase().includes(query);
        const descMatch = feat.description?.toLowerCase().includes(query);
        if (!nameMatch && !idMatch && !descMatch) return false;
      }

      // 2. Filter by status state
      if (selectedState !== 'all' && feat.state !== selectedState) {
        return false;
      }

      return true;
    });
  }, [searchQuery, selectedState]);

  const latestUpdatedTimestamp = useMemo(() => {
    if (displayedStatuses.length === 0) return '';
    return displayedStatuses[0].last_updated;
  }, [displayedStatuses]);

  return (
    <div>
      <Header
        tenant={tenant}
        locale={locale}
        product={product}
        onRefresh={() => loadData(false)}
        isRefreshing={isRefreshing}
      />

      <main className="status-container">
        {/* Active Incident Announcement Banner (operator-controlled) */}
        <AnnouncementBanner incidents={incidents} />

        {/* System Overview Hero Banner (with automated pre-made check-down messages) */}
        <SystemBanner
          overallState={overallState}
          productFilter={product}
          tenant={tenant}
          lastUpdated={latestUpdatedTimestamp}
          affectedFeatures={affectedFeatures}
        />

        {/* Global Key Metrics */}
        <MetricsOverview statuses={displayedStatuses} />

        {/* In-Page Feature Filter Toolbar & Product Switcher */}
        <FilterToolbar
          tenant={tenant}
          locale={locale}
          currentProduct={product}
          allProducts={allProducts}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          selectedState={selectedState}
          onStateSelect={setSelectedState}
          featureCounts={featureCounts}
        />

        {/* Product & Feature Cards */}
        <section className="products-section" aria-label="Product health list">
          {displayedStatuses.length === 0 && !loading && (
            <div className="empty-state">
              <h3>No Product Status Records Found</h3>
              <p>There are no monitored products currently registered.</p>
            </div>
          )}

          {displayedStatuses.map((status) => {
            const filteredFeatures = getFilteredFeatures(status.features);
            const linkedIncidents = incidents.filter(
              (inc) =>
                inc.active &&
                (inc.products?.includes(status.product) ||
                  inc.product === status.product ||
                  (inc.product === '*' && (!inc.products || inc.products.length === 0)))
            );

            return (
              <ProductStatusCard
                key={`${status.tenant}-${status.product}`}
                status={status}
                filteredFeatures={filteredFeatures}
                linkedIncidents={linkedIncidents}
              />
            );
          })}
        </section>

        <Footer />
      </main>
    </div>
  );
};
