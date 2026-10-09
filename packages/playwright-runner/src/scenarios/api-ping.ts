import type { Browser } from "playwright";
import type { RunCheckResponse, ScenarioContext } from "../types.js";

export async function runApiPingScenario(
  browser: Browser,
  ctx: ScenarioContext
): Promise<RunCheckResponse> {
  const startTime = Date.now();
  const requestContext = await browser.newContext();

  try {
    const timeout = ctx.timeoutMs || 10000;
    const method = ctx.parameters["method"] || "GET";
    const res = await requestContext.request.fetch(ctx.target, {
      method,
      timeout,
    });

    const latencyMs = Date.now() - startTime;
    const status = res.status();

    if (status >= 500) {
      return {
        success: false,
        state: "outage",
        latencyMs,
        message: `API check failed with HTTP ${status}`,
        details: { status },
      };
    }

    if (status >= 400) {
      return {
        success: false,
        state: "degraded",
        latencyMs,
        message: `API check returned HTTP ${status}`,
        details: { status },
      };
    }

    return {
      success: true,
      state: "operational",
      latencyMs,
      message: `API check nominal (${latencyMs}ms, HTTP ${status})`,
      details: { status },
    };
  } catch (err: unknown) {
    const latencyMs = Date.now() - startTime;
    const errorMsg = err instanceof Error ? err.message : String(err);
    return {
      success: false,
      state: "outage",
      latencyMs,
      message: `API check error: ${errorMsg}`,
      details: { error: errorMsg },
    };
  } finally {
    await requestContext.close().catch(() => {});
  }
}
