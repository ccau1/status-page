import { describe, it, expect } from 'vitest';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { StatusPage } from './StatusPage';
import { SystemBanner } from '../components/SystemBanner';
import { UptimeBar } from '../components/UptimeBar';
import { AnnouncementBanner } from '../components/AnnouncementBanner';
import { Incident } from '../types/status';

describe('SystemBanner component', () => {
  it('renders operational banner correctly', () => {
    render(
      <SystemBanner
        overallState="operational"
        tenant="acme"
        lastUpdated="2026-09-27T10:00:00Z"
      />
    );
    expect(screen.getByText('All Systems Operational')).toBeDefined();
    expect(screen.getByText(/OPERATIONAL/)).toBeDefined();
  });

  it('renders pre-made message when checks are degraded with affected features', () => {
    render(
      <SystemBanner
        overallState="degraded"
        tenant="acme"
        lastUpdated="2026-09-27T10:00:00Z"
        affectedFeatures={['Dynamic Tax Jurisdiction Engine (billing-engine)']}
      />
    );
    expect(screen.getByText('Degraded Performance Detected on Monitored Checks')).toBeDefined();
    expect(screen.getByText(/Automated health probes detected elevated response times on Dynamic Tax Jurisdiction Engine/)).toBeDefined();
  });

  it('renders product-scoped degraded banner', () => {
    render(
      <SystemBanner
        overallState="degraded"
        productFilter="billing-engine"
        tenant="acme"
        lastUpdated="2026-09-27T10:00:00Z"
      />
    );
    expect(screen.getByText('BILLING-ENGINE is experiencing degraded performance')).toBeDefined();
  });
});

describe('AnnouncementBanner component', () => {
  const sampleIncidents: Incident[] = [
    {
      id: 'inc-test-1',
      tenant: '*',
      product: 'checkout',
      title: 'Active Payment Gateway Latency',
      severity: 'major',
      state: 'investigating',
      message: 'Engineering is investigating elevated latency.',
      active: true,
      created_at: '2026-09-27T10:00:00Z',
      updated_at: '2026-09-27T10:15:00Z',
      updates: [
        {
          timestamp: '2026-09-27T10:00:00Z',
          state: 'investigating',
          message: 'Monitoring detected threshold breach.',
        },
        {
          timestamp: '2026-09-27T10:15:00Z',
          state: 'identified',
          message: 'Identified root cause in upstream router.',
        },
      ],
    },
  ];

  it('renders incident announcement with severity and title', () => {
    render(<AnnouncementBanner incidents={sampleIncidents} />);
    expect(screen.getByText('Active Payment Gateway Latency')).toBeDefined();
    expect(screen.getByText('MAJOR')).toBeDefined();
    expect(screen.getByText('INVESTIGATING')).toBeDefined();
  });

  it('expands and collapses timeline progress updates', () => {
    render(<AnnouncementBanner incidents={sampleIncidents} />);
    const timelineBtn = screen.getByRole('button', { name: /View Progress Updates/i });
    fireEvent.click(timelineBtn);

    expect(screen.getByText('Identified root cause in upstream router.')).toBeDefined();

    // Collapse
    fireEvent.click(timelineBtn);
    expect(screen.queryByText('Identified root cause in upstream router.')).toBeNull();
  });

  it('dismisses banner on close button click', () => {
    render(<AnnouncementBanner incidents={sampleIncidents} />);
    const closeBtn = screen.getByRole('button', { name: /Dismiss banner/i });
    fireEvent.click(closeBtn);

    expect(screen.queryByText('Active Payment Gateway Latency')).toBeNull();
  });
});

describe('UptimeBar component', () => {
  it('renders uptime history ticks', () => {
    const { container } = render(<UptimeBar currentState="operational" days={30} />);
    const ticks = container.querySelectorAll('.uptime-tick');
    expect(ticks.length).toBe(30);
  });
});

describe('StatusPage in-page feature filtering', () => {
  it('filters features when searching text', async () => {
    render(
      <MemoryRouter initialEntries={['/default/en/status']}>
        <Routes>
          <Route path="/:tenant/:locale/status" element={<StatusPage />} />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('API Gateway Routing')).toBeDefined();
    expect(screen.getByText('Recurring Invoice Generator')).toBeDefined();

    const searchInput = screen.getByPlaceholderText(/Filter features/i);
    fireEvent.change(searchInput, { target: { value: 'gateway' } });

    expect(screen.getByText('API Gateway Routing')).toBeDefined();
    expect(screen.queryByText('Recurring Invoice Generator')).toBeNull();
  });

  it('filters features by health state', async () => {
    render(
      <MemoryRouter initialEntries={['/default/en/status']}>
        <Routes>
          <Route path="/:tenant/:locale/status" element={<StatusPage />} />
        </Routes>
      </MemoryRouter>
    );

    const degradedButton = screen.getByRole('button', { name: /Degraded/i });
    fireEvent.click(degradedButton);

    expect(screen.getByText('Dynamic Tax Jurisdiction Engine')).toBeDefined();
    expect(screen.queryByText('OAuth 2.0 & OIDC Provider')).toBeNull();
  });
});
