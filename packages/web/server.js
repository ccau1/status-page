import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import express from 'express';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isProduction = process.env.NODE_ENV === 'production';
const port = process.env.PORT || 3000;
const base = process.env.BASE || '/';

export async function createStatusWebServer() {
  const app = express();

  let vite;
  if (!isProduction) {
    const { createServer } = await import('vite');
    vite = await createServer({
      root: __dirname,
      server: { middlewareMode: true },
      appType: 'custom',
      base,
    });
    app.use(vite.middlewares);
  } else {
    const compression = (await import('compression')).default;
    const sirv = (await import('sirv')).default;
    app.use(compression());
    app.use(base, sirv(path.resolve(__dirname, 'dist/client'), { extensions: [] }));
  }

  // Reverse proxy API and Auth requests directly to backend
  const apiBackend = process.env.API_BACKEND_URL || process.env.API_BASE_URL || 'http://api:8080';
  app.use(['/api', '/auth'], async (req, res) => {
    try {
      const targetUrl = new URL(req.originalUrl, apiBackend);
      const headers = { ...req.headers, host: targetUrl.host };
      delete headers['content-length'];

      const fetchOpts = {
        method: req.method,
        headers,
        redirect: 'manual',
      };

      if (req.method !== 'GET' && req.method !== 'HEAD') {
        const chunks = [];
        for await (const chunk of req) {
          chunks.push(chunk);
        }
        fetchOpts.body = Buffer.concat(chunks);
      }

      const response = await fetch(targetUrl.toString(), fetchOpts);

      res.status(response.status);
      response.headers.forEach((val, key) => {
        res.setHeader(key, val);
      });

      const buffer = await response.arrayBuffer();
      res.send(Buffer.from(buffer));
    } catch (err) {
      console.error('[Web Proxy Error]', err.message);
      res.status(502).json({ error: 'Backend service unavailable: ' + err.message });
    }
  });

  // Handle all SSR requests
  app.use('*', async (req, res, next) => {
    const rawUrl = req.originalUrl;
    const url = rawUrl.startsWith('/') ? rawUrl : '/' + rawUrl;

    try {
      let template;
      let render;

      if (!isProduction) {
        // Read index.html directly and apply Vite HMR transforms
        template = await fs.readFile(path.resolve(__dirname, 'index.html'), 'utf-8');
        template = await vite.transformIndexHtml(url, template);
        render = (await vite.ssrLoadModule('/src/entry-server.tsx')).render;
      } else {
        template = await fs.readFile(path.resolve(__dirname, 'dist/client/index.html'), 'utf-8');
        render = (await import('./dist/server/entry-server.js')).render;
      }

      const rendered = render(url);
      const apiBaseScript = `<script>window.__API_BASE_URL__ = "";</script>`;
      let html = template.replace('<!--ssr-outlet-->', rendered.html ?? '');
      html = html.replace('</head>', `${apiBaseScript}</head>`);

      res.status(200).set({ 'Content-Type': 'text/html' }).end(html);
    } catch (e) {
      if (vite) {
        vite.ssrFixStacktrace(e);
      }
      console.error('[SSR Error]', e.stack);
      res.status(500).end(e.stack);
    }
  });

  return { app };
}

createStatusWebServer().then(({ app }) => {
  app.listen(port, () => {
    console.log(`[Web SSR] Status Page frontend running at http://localhost:${port}`);
  });
});
