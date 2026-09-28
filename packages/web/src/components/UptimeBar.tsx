import React from 'react';
import { StatusState } from '../types/status';

interface UptimeBarProps {
  currentState: StatusState;
  days?: number;
}

export const UptimeBar: React.FC<UptimeBarProps> = ({ currentState, days = 60 }) => {
  // Generate procedural past ticks with high fidelity
  const ticks = React.useMemo(() => {
    const list: { day: number; state: StatusState; label: string }[] = [];
    for (let i = days; i >= 1; i--) {
      let tickState: StatusState = 'operational';
      let label = `${i} days ago: 100% operational`;

      // Today is represented by current state
      if (i === 1) {
        tickState = currentState;
        label = `Today: ${currentState.toUpperCase()}`;
      } else if (currentState === 'degraded' && (i === 4 || i === 18)) {
        tickState = 'degraded';
        label = `${i} days ago: Minor performance degradation`;
      } else if (currentState === 'outage' && i === 2) {
        tickState = 'outage';
        label = `${i} days ago: Incident reported`;
      }

      list.push({ day: i, state: tickState, label });
    }
    return list;
  }, [currentState, days]);

  return (
    <div className="uptime-section">
      <div className="uptime-label-row">
        <span>{days} days ago</span>
        <span style={{ color: 'var(--color-operational)' }}>99.98% uptime</span>
        <span>Today</span>
      </div>
      <div className="uptime-bar" role="group" aria-label="Uptime history">
        {ticks.map((t) => (
          <div
            key={t.day}
            className={`uptime-tick state-${t.state}`}
            title={t.label}
          />
        ))}
      </div>
    </div>
  );
};
