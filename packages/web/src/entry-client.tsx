import React from 'react';
import { hydrateRoot, createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router';
import { App } from './App';

const container = document.getElementById('root')!;

if (container.hasChildNodes()) {
  hydrateRoot(
    container,
    <React.StrictMode>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </React.StrictMode>
  );
} else {
  createRoot(container).render(
    <React.StrictMode>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </React.StrictMode>
  );
}
