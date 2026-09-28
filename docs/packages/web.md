# `packages/web` — Status Frontend & Admin Portal

The web package is a modern, responsive React 19 application rendered with **Vite Server-Side Rendering (SSR)**. It serves both the public-facing status pages and the internal administrative incident control console.

---

## 🌐 Dynamic Routing Structure

The application supports the following dynamic path hierarchy:

```text
/{tenant?}/{locale?}/status/{product?}
/admin
```

### Route Patterns & Mapping:
| URL Pattern | Example | Description |
|---|---|---|
| `/:tenant/:locale/status/:product` | `/acme/en/status/cloud-api` | Specific product status under a specific tenant and language |
| `/:tenant/:locale/status` | `/acme/es/status` | All products for tenant `acme` in Spanish (`es`) |
| `/:tenant/status/:product` | `/enterprise/status/billing` | Defaults locale to `en` |
| `/:tenant/status` | `/enterprise/status` | All products for tenant `enterprise` (default locale `en`) |
| `/status/:product` | `/status/cloud-api` | Default tenant (`default`), default locale (`en`), specific product |
| `/status` | `/status` | Default tenant and locale, all products |
| `/` | `/` | Redirects to `/default/en/status` |
| `/admin` | `/admin` | Internal Incident Management Center (SSO / Password login) |

---

## 🛡 Internal Incident Management Portal (`/admin`)

The `/admin` route provides an operations portal for incident responders and platform administrators:

### 1. Tabbed Authentication Screen (When Unauthenticated)
* **Sign In Tab:**
  * One-click **Enterprise SSO** (Microsoft Entra ID / Okta / Local Dev SSO).
  * Direct **Username & Password** login form.
* **Register Account Tab:**
  * Self-service registration form (Username, Email, Display Name, Password) delegating to the stateless gRPC auth service.
* **Forgot Password Tab:**
  * Request signed password reset token via email.
  * Enter reset token and new password to reset and automatically log in.

### 2. Incident Control Center (When Authenticated)
* **Broadcast New Incidents:** Form allowing selection of Title, Product scope (*Global* or specific product), Severity (*Critical*, *Major*, *Minor*, *Maintenance*, *Info*), Initial investigation state, and message.
* **Live Incident Management:**
  * View active incident cards.
  * **`+ Post Update`**: Adds a new progress step to the chronological timeline with an updated investigation state (`investigating` $\rightarrow$ `identified` $\rightarrow$ `monitoring` $\rightarrow$ `resolved`).
  * **`Resolve & Remove`**: Deactivates the banner and archives it from public status dashboards.

---

## 📢 Public Incident & Check-Down Banners

1. **Active Announcement Banner (`AnnouncementBanner`):**
   * Prominently displayed across the top of public dashboards when an operator broadcasts an issue.
   * Colored by severity level with pulsing investigation stage indicator.
   * Includes interactive **"View Progress Updates (N) ▼"** toggle to inspect the chronological timeline of updates logged by the response team.
   * Dismissible by the visitor via `✕`.

2. **Automated Pre-Made Check-Down Messaging (`SystemBanner`):**
   * If individual health checks fail without an active operator announcement:
     * System aggregates failing checks across active products.
     * Generates a formulated message:
       * *Degraded:* `"Automated health probes detected elevated response times on [Feature Name] ([Product]). Core traffic remains online while systems stabilize."*
       * *Outage:* `"Automated checks reported failing responses on [Feature Name] ([Product]). Incident response procedures are actively engaged."*

---

## 🔍 In-Page Feature Filtering

Rather than triggering full-page navigations or network re-fetches, feature drill-downs operate as an **in-page client filter**:
* **Real-time Keyword Search:** Instant search input filtering features by name, identifier, and description.
* **Health State Filter Chips:** Quick toggles to inspect `Operational`, `Degraded`, or `Outages` only, with live count badges.
* **Product Tabs:** Direct switching between products while preserving the active in-page search criteria.

---

## 🎨 Visual Design & Vanilla CSS

Built with a bespoke **Obsidian Glassmorphism** design system:
* **Backgrounds:** Multi-layered radial gradients over `#070a12`.
* **Cards & Surfaces:** Translucent panels with subtle border glows and `backdrop-filter: blur(14px)`.
* **Live Indicators:** CSS-animated pulsing rings indicating real-time system connectivity.
* **Procedural Uptime Tracks:** 60-day interactive tick bar displaying historical uptime percentages with hover metadata.
* **Zero Tailwind Dependency:** 100% Vanilla CSS in `src/styles/main.css` for maximum maintainability and zero build bloat.

---

## 🖥 SSR Server Architecture (`server.js`)

`packages/web/server.js` provides dual-mode execution:
1. **Development Mode (`npm run dev`):**
   * Uses Vite in middleware mode (`createServer({ server: { middlewareMode: true } })`).
   * Hot Module Replacement (HMR) active with polling support for Docker bind mounts.
   * `vite.ssrLoadModule('/src/entry-server.tsx')` dynamically loads and executes server components on each request.
2. **Production Mode (`npm run preview`):**
   * Serves optimized static assets via `sirv` and `compression`.
   * Pre-loads the pre-built SSR bundle (`dist/server/entry-server.js`) to render initial HTML before client hydration.
