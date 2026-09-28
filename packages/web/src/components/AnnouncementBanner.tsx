import React, { useState } from 'react';
import { Incident } from '../types/status';

interface AnnouncementBannerProps {
  incidents: Incident[];
}

export const AnnouncementBanner: React.FC<AnnouncementBannerProps> = ({ incidents }) => {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [dismissedIds, setDismissedIds] = useState<string[]>([]);

  const activeIncidents = incidents.filter(
    (inc) => inc.active && !dismissedIds.includes(inc.id)
  );

  if (activeIncidents.length === 0) {
    return null;
  }

  const handleDismiss = (id: string) => {
    setDismissedIds((prev) => [...prev, id]);
  };

  const toggleExpand = (id: string) => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  return (
    <div className="announcements-container" role="region" aria-label="Incident Announcements">
      {activeIncidents.map((inc) => {
        const isExpanded = expandedId === inc.id;
        const formattedTime = new Date(inc.updated_at).toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
        });

        return (
          <div key={inc.id} className={`announcement-card severity-${inc.severity}`}>
            <div className="announcement-header">
              <div className="announcement-badge-group">
                <span className={`announcement-severity-pill severity-${inc.severity}`}>
                  {inc.severity.toUpperCase()}
                </span>
                <span className={`announcement-state-pill state-${inc.state}`}>
                  <span className="pulse-dot" style={{ width: '6px', height: '6px' }}></span>
                  {inc.state.toUpperCase()}
                </span>
                {/* Linked Products */}
                {inc.products && inc.products.length > 0 ? (
                  inc.products.map((p) => (
                    <span key={p} className="announcement-product-tag" title="Linked Product">
                      📦 {p}
                    </span>
                  ))
                ) : inc.product ? (
                  <span className="announcement-product-tag" title="Linked Product">
                    📦 {inc.product}
                  </span>
                ) : (
                  <span className="announcement-product-tag" style={{ color: 'var(--text-dim)' }}>
                    🌐 Global
                  </span>
                )}

                {/* Linked Checks */}
                {inc.check_ids && inc.check_ids.map((cid) => (
                  <span key={cid} className="announcement-product-tag" style={{ borderColor: 'rgba(59, 130, 246, 0.4)', color: '#93c5fd' }} title="Linked Health Check">
                    🔍 {cid}
                  </span>
                ))}

                {/* Linked Features */}
                {inc.features && inc.features.map((fid) => (
                  <span key={fid} className="announcement-product-tag" style={{ borderColor: 'rgba(168, 85, 247, 0.4)', color: '#d8b4fe' }} title="Linked Feature">
                    ⚡ {fid}
                  </span>
                ))}
              </div>

              <button
                className="announcement-close-btn"
                onClick={() => handleDismiss(inc.id)}
                title="Dismiss banner"
                aria-label="Dismiss banner"
              >
                ✕
              </button>
            </div>

            <div className="announcement-body">
              <h2 className="announcement-title">{inc.title}</h2>
              <p className="announcement-message">{inc.message}</p>
            </div>

            <div className="announcement-footer">
              <span className="announcement-time">
                Latest update: {formattedTime}
              </span>

              {inc.updates && inc.updates.length > 0 && (
                <button
                  className="announcement-timeline-toggle"
                  onClick={() => toggleExpand(inc.id)}
                  aria-expanded={isExpanded}
                >
                  {isExpanded ? 'Hide Timeline ▲' : `View Progress Updates (${inc.updates.length}) ▼`}
                </button>
              )}
            </div>

            {/* Expandable Incident Progress Timeline */}
            {isExpanded && inc.updates && (
              <div className="announcement-timeline">
                <div className="timeline-track">
                  {inc.updates.map((upd, idx) => (
                    <div key={idx} className="timeline-entry">
                      <div className="timeline-marker"></div>
                      <div className="timeline-content">
                        <div className="timeline-header">
                          <span className={`timeline-state-tag state-${upd.state}`}>
                            {upd.state.toUpperCase()}
                          </span>
                          <span className="timeline-date">
                            {new Date(upd.timestamp).toLocaleString([], {
                              month: 'short',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                            })}
                          </span>
                        </div>
                        <p className="timeline-msg">{upd.message}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
