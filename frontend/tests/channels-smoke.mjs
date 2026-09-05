import assert from "node:assert/strict";

// Run against a dev/preview server; every API request is intercepted.
// PLAYWRIGHT_MODULE may point to an existing Playwright installation.
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE || "playwright");
const browser = await chromium.launch({ headless: true, channel: process.env.BROWSER_CHANNEL || "chrome" });
const origin = process.env.FRONTEND_URL || "http://127.0.0.1:5174";
try {
  for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: 900 } });
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    let connections = [];
    let policies = [];
    const provider = {
      id: "provider-test", name: "napcatqq", display_name: "NapCatQQ",
      inbound_modes: ["http_webhook"], outbound_modes: ["send_message"],
      metadata: {}, status: "active"
    };
    await page.addInitScript(() => {
      localStorage.setItem("freedinner.accessToken", "mock-token");
      localStorage.setItem("freedinner.locale", "zh-CN");
    });
    await page.route("**/api/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      const method = route.request().method();
      let data = [];
      if (path.endsWith("/me")) data = { id: "test-user", username: "test-user" };
      if (path.endsWith("/channel-providers")) data = [provider];
      if (path.endsWith("/channel-connections")) {
        if (method === "POST") {
          connections = [{
            ...route.request().postDataJSON(), id: "connection-test",
            status: "active", has_config: true, created_at: "2026-09-05T00:00:00Z"
          }];
          data = connections[0];
        } else data = connections;
      }
      if (path.endsWith("/policies")) {
        if (method === "PATCH") {
          policies = [{ ...route.request().postDataJSON(), id: "policy-test", metadata: {} }];
          data = policies[0];
        } else data = policies;
      }
      await route.fulfill({ json: { data, error: null } });
    });
    await page.goto(`${origin}/app/channels`);
    await page.getByRole("button", { name: "创建连接", exact: true }).click();
    await page.locator("form").getByRole("button", { name: "创建连接", exact: true }).click();
    await page.waitForURL("**/connection-test/sessions");
    assert.equal(connections.length, 1);
    await page.goto(`${origin}/app/channels/connection-test/policies`);
    await page.getByRole("button", { name: "创建策略", exact: true }).click();
    const form = page.locator("form");
    await form.locator("select").nth(1).selectOption("mention_or_keyword");
    await form.getByPlaceholder(/关键词/).fill("天气，新闻\n提醒");
    await form.locator('input[type="number"]').fill("0");
    await form.getByRole("button", { name: "保存策略", exact: true }).click();
    const policyList = page.locator("section").filter({ has: page.getByRole("heading", { name: "当前策略", exact: true }) }).last();
    await policyList.getByRole("button", { name: "编辑", exact: true }).click();
    assert.equal(await form.locator("select").nth(1).inputValue(), "mention_or_keyword");
    assert.deepEqual(policies[0].trigger_keywords, ["天气", "新闻", "提醒"]);
    assert.equal(policies[0].rate_limit_per_minute, 0);
    await form.locator("select").nth(1).selectOption("disabled");
    await form.getByRole("button", { name: "保存策略", exact: true }).click();
    await form.waitFor({ state: "hidden" });
    assert.equal(policies[0].mode, "disabled");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
    assert.deepEqual(errors, []);
    if (process.env.SCREENSHOT_DIR) {
      await page.screenshot({ path: `${process.env.SCREENSHOT_DIR}/channels-${width}.png`, fullPage: true });
    }
    console.log(`PASS channels ${width}px: empty creation, combined policy, edit, zero rate, disabled, no overflow`);
    await page.close();
  }
} finally {
  await browser.close();
}
