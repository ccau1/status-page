import type { Browser } from "playwright";
import type { RunCheckResponse, ScenarioContext } from "../types.js";

export async function runPageLoadScenario(
  browser: Browser,
  ctx: ScenarioContext
): Promise<RunCheckResponse> {
  const startTime = Date.now();
  const context = await browser.newContext({
    ignoreHTTPSErrors: ctx.parameters["ignore_https_errors"] === "true",
  });
  const page = await context.newPage();

  try {
    const timeout = ctx.timeoutMs || 15000;
    page.setDefaultTimeout(timeout);

    const waitUntil = (ctx.parameters["wait_until"] as "load" | "domcontentloaded" | "networkidle") || "domcontentloaded";

    const response = await page.goto(ctx.target, {
      waitUntil,
      timeout,
    });

    const latencyMs = Date.now() - startTime;

    if (!response) {
      return {
        success: false,
        state: "outage",
        latencyMs,
        message: `Failed to load page: no response from ${ctx.target}`,
      };
    }

    const httpStatus = response.status();
    if (httpStatus >= 500) {
      return {
        success: false,
        state: "outage",
        latencyMs,
        message: `HTTP ${httpStatus} returned from ${ctx.target}`,
        details: { httpStatus },
      };
    }

    if (httpStatus >= 400) {
      return {
        success: false,
        state: "degraded",
        latencyMs,
        message: `Client error HTTP ${httpStatus} from ${ctx.target}`,
        details: { httpStatus },
      };
    }

    // Optional selector check
    const selector = ctx.parameters["selector"];
    if (selector) {
      await page.waitForSelector(selector, { state: "visible", timeout });
    }

    // Optional expected text check
    const expectedText = ctx.parameters["expected_text"];
    if (expectedText) {
      const content = await page.content();
      if (!content.includes(expectedText)) {
        return {
          success: false,
          state: "degraded",
          latencyMs,
          message: `Expected text "${expectedText}" not found in page content`,
          details: { httpStatus },
        };
      }
    }

    const title = await page.title();

    // Check degraded latency threshold if configured
    let state: "operational" | "degraded" = "operational";
    const degradedLatencyThreshold = ctx.parameters["degraded_latency_ms"]
      ? parseInt(ctx.parameters["degraded_latency_ms"], 10)
      : 0;

    if (degradedLatencyThreshold > 0 && latencyMs > degradedLatencyThreshold) {
      state = "degraded";
    }

    return {
      success: true,
      state,
      latencyMs,
      message: `Page load nominal (${latencyMs}ms, HTTP ${httpStatus})`,
      details: {
        httpStatus,
        title,
        url: page.url(),
      },
    };
  } catch (err: unknown) {
    const latencyMs = Date.now() - startTime;
    const errorMsg = err instanceof Error ? err.message : String(err);
    return {
      success: false,
      state: "outage",
      latencyMs,
      message: `Playwright check error: ${errorMsg}`,
      details: { error: errorMsg },
    };
  } finally {
    await context.close().catch(() => {});
  }
}
