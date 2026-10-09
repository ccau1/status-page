import express from "express";
import { PlaywrightRunner } from "./runner.js";

const app = express();
const port = parseInt(process.env.PORT || "8088", 10);

app.use(express.json());

const runner = new PlaywrightRunner();

// Readiness and liveness probe for EKS / ECS
app.get("/health", (_req, res) => {
  res.status(200).json({ status: "healthy", service: "playwright-runner" });
});

app.post("/run", async (req, res) => {
  const { scenario, target, timeoutMs, parameters } = req.body || {};

  if (!target || typeof target !== "string") {
    res.status(400).json({
      success: false,
      state: "outage",
      latencyMs: 0,
      message: "Missing required parameter 'target'",
    });
    return;
  }

  try {
    const result = await runner.run({
      scenario,
      target,
      timeoutMs: typeof timeoutMs === "number" ? timeoutMs : undefined,
      parameters: parameters && typeof parameters === "object" ? parameters : {},
    });
    res.status(200).json(result);
  } catch (err: unknown) {
    const errorMsg = err instanceof Error ? err.message : String(err);
    res.status(500).json({
      success: false,
      state: "outage",
      latencyMs: 0,
      message: `Internal runner error: ${errorMsg}`,
    });
  }
});

const server = app.listen(port, "0.0.0.0", async () => {
  console.log(`[playwright-runner] Service listening on http://0.0.0.0:${port}`);
  // Eagerly pre-warm Chromium
  try {
    await runner.init();
  } catch (err) {
    console.warn("[playwright-runner] Warning: Lazy browser initialization on first request:", err);
  }
});

async function shutdown(signal: string) {
  console.log(`[playwright-runner] Received ${signal}, shutting down gracefully...`);
  server.close(async () => {
    await runner.close();
    process.exit(0);
  });
}

process.on("SIGTERM", () => shutdown("SIGTERM"));
process.on("SIGINT", () => shutdown("SIGINT"));
