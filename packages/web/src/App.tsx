import React from 'react';
import { Routes, Route, Navigate } from 'react-router';
import { StatusPage } from './routes/StatusPage';
import { AdminPage } from './routes/AdminPage';
import './styles/main.css';

export const App: React.FC = () => {
  return (
    <Routes>
      {/* Internal Admin Incident Management Route */}
      <Route path="/admin" element={<AdminPage />} />

      {/* Dynamic routes supporting /{tenant?}/{locale?}/status/{product?} */}
      <Route path="/:tenant/:locale/status/:product" element={<StatusPage />} />
      <Route path="/:tenant/:locale/status" element={<StatusPage />} />
      <Route path="/:tenant/status/:product" element={<StatusPage />} />
      <Route path="/:tenant/status" element={<StatusPage />} />
      <Route path="/status/:product" element={<StatusPage />} />
      <Route path="/status" element={<StatusPage />} />

      {/* Default index redirect */}
      <Route path="/" element={<Navigate to="/status" replace />} />
      <Route path="*" element={<Navigate to="/status" replace />} />
    </Routes>
  );
};
