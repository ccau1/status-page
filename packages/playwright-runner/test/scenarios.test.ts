import test from "node:test";
import assert from "node:assert/strict";
import { getScenario, registerScenario } from "../src/scenarios/index.js";
import type { ScenarioContext } from "../src/types.js";

test("Scenario Registry resolves default and custom scenarios", () => {
  const defaultHandler = getScenario();
  assert.ok(defaultHandler, "should return a default handler");

  const pageLoadHandler = getScenario("page-load");
  assert.strictEqual(pageLoadHandler, defaultHandler);

  registerScenario("custom-test", async (_browser, ctx: ScenarioContext) => {
    return {
      success: true,
      state: "operational",
      latencyMs: 12,
      message: `Custom check on ${ctx.target}`,
    };
  });

  const customHandler = getScenario("custom-test");
  assert.ok(customHandler);
});
