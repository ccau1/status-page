import type { Browser } from "playwright";
import type { RunCheckResponse, ScenarioContext } from "../types.js";
import { runPageLoadScenario } from "./page-load.js";
import { runApiPingScenario } from "./api-ping.js";

export type ScenarioHandler = (
  browser: Browser,
  ctx: ScenarioContext
) => Promise<RunCheckResponse>;

const scenarios: Record<string, ScenarioHandler> = {
  "page-load": runPageLoadScenario,
  "default": runPageLoadScenario,
  "api-ping": runApiPingScenario,
};

export function getScenario(name?: string): ScenarioHandler {
  if (!name || !(name in scenarios)) {
    return scenarios["default"];
  }
  return scenarios[name];
}

export function registerScenario(name: string, handler: ScenarioHandler): void {
  scenarios[name] = handler;
}
