import { chromium, type Browser } from "playwright";
import { getScenario } from "./scenarios/index.js";
import type { RunCheckRequest, RunCheckResponse } from "./types.js";

export class PlaywrightRunner {
  private browser: Browser | null = null;
  private isLaunching = false;

  async init(): Promise<void> {
    if (this.browser && this.browser.isConnected()) {
      return;
    }
    if (this.isLaunching) {
      while (this.isLaunching) {
        await new Promise((r) => setTimeout(r, 50));
      }
      return;
    }

    this.isLaunching = true;
    try {
      this.browser = await chromium.launch({
        headless: true,
        args: [
          "--no-sandbox",
          "--disable-setuid-sandbox",
          "--disable-dev-shm-usage",
          "--disable-gpu",
        ],
      });
      console.log("[PlaywrightRunner] Chromium browser launched successfully.");
    } finally {
      this.isLaunching = false;
    }
  }

  async run(req: RunCheckRequest): Promise<RunCheckResponse> {
    if (!this.browser || !this.browser.isConnected()) {
      await this.init();
    }

    if (!this.browser) {
      return {
        success: false,
        state: "outage",
        latencyMs: 0,
        message: "Failed to initialize Playwright browser instance",
      };
    }

    const handler = getScenario(req.scenario);
    return handler(this.browser, {
      target: req.target,
      timeoutMs: req.timeoutMs || 15000,
      parameters: req.parameters || {},
    });
  }

  async close(): Promise<void> {
    if (this.browser) {
      console.log("[PlaywrightRunner] Closing browser instance...");
      await this.browser.close().catch(() => {});
      this.browser = null;
    }
  }
}
