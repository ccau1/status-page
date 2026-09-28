import React from 'react';
import { StatusState } from '../types/status';

interface SystemBannerProps {
  overallState: StatusState;
  productFilter?: string;
  tenant: string;
  lastUpdated: string;
  affectedFeatures?: string[];
}

export const SystemBanner: React.FC<SystemBannerProps> = ({
  overallState,
  productFilter,
  tenant,
  lastUpdated,
  affectedFeatures = [],
}) => {
  const formattedAffected = affectedFeatures.length > 0
    ? affectedFeatures.slice(0, 3).join(', ') + (affectedFeatures.length > 3 ? ` and ${affectedFeatures.length - 3} more` : '')
    : '';

  const getBannerTitle = () => {
    if (productFilter) {
      switch (overallState) {
        case 'operational': return `${productFilter.toUpperCase()} is fully operational`;
        case 'degraded': return `${productFilter.toUpperCase()} is experiencing degraded performance`;
        case 'outage': return `${productFilter.toUpperCase()} is experiencing a service outage`;
        case 'maintenance': return `${productFilter.toUpperCase()} is under scheduled maintenance`;
        default: return `${productFilter.toUpperCase()} status is unknown`;
      }
    }
    switch (overallState) {
      case 'operational': return 'All Systems Operational';
      case 'degraded': return 'Degraded Performance Detected on Monitored Checks';
      case 'outage': return 'Critical Outage Detected on Core Capabilities';
      case 'maintenance': return 'Scheduled System Maintenance in Progress';
      default: return 'System Status Unknown';
    }
  };

  const getBannerSubtitle = () => {
    switch (overallState) {
      case 'operational':
        return 'Continuous monitoring across all cluster nodes. 100% of health check probes passing without anomalies.';
      case 'degraded':
        if (formattedAffected) {
          return `Automated health probes detected elevated response times on ${formattedAffected}. Core traffic remains online while systems stabilize.`;
        }
        return 'Engineering teams are currently monitoring elevated latency or failure rates on isolated endpoints.';
      case 'outage':
        if (formattedAffected) {
          return `Automated checks reported failing responses on ${formattedAffected}. Incident response procedures are actively engaged.`;
        }
        return 'Incident responders are actively resolving core component disruptions across cluster nodes.';
      case 'maintenance':
        return 'Planned routine maintenance is underway. Minor intermittent connection pauses may occur.';
      default:
        return 'Gathering metrics from health check worker daemons...';
    }
  };

  const formattedTime = lastUpdated
    ? new Date(lastUpdated).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    : 'Just now';

  return (
    <div className={`hero-banner state-${overallState}`}>
      <div className="hero-content">
        <div className="hero-title-group">
          <h1>{getBannerTitle()}</h1>
          <p>{getBannerSubtitle()}</p>
        </div>
        <div className="hero-badge-wrap">
          <div className={`status-pill-large state-${overallState}`}>
            <span className="pulse-dot"></span>
            {overallState.toUpperCase()}
          </div>
          <span style={{ fontSize: '0.8rem', color: 'var(--text-dim)' }}>
            Updated {formattedTime}
          </span>
        </div>
      </div>
    </div>
  );
};
