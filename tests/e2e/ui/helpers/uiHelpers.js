const { expect } = require("@playwright/test");

function uniqueSuffix() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

async function login(page) {
  await page.goto("/");

  const loginButton = page.getByRole("button", { name: /^login$/i });
  if (await loginButton.isVisible()) {
    await page.getByPlaceholder("admin@example.com").fill("admin@example.com");
    await page.getByPlaceholder("admin123").fill("admin123");
    await loginButton.click();
  }

  await expect(page.getByRole("heading", { name: /Enterprise Asset Monitoring/i })).toBeVisible();
}

async function gotoNav(page, label) {
  const navButton = page.getByRole("button", { name: new RegExp(`^${label}$`, "i") });
  await expect(navButton).toBeVisible({ timeout: 15000 });
  await navButton.click();
  await page.waitForLoadState("networkidle");
}

async function clickIfVisible(locator) {
  if (!locator) {
    return false;
  }

  const count = await locator.count();
  if (count === 0) {
    return false;
  }

  const first = locator.first();
  if (await first.isVisible()) {
    await first.click();
    return true;
  }

  return false;
}

async function fillIfVisible(locator, value) {
  if (!locator) {
    return false;
  }

  const count = await locator.count();
  if (count === 0) {
    return false;
  }

  const first = locator.first();
  if (await first.isVisible()) {
    await first.fill(value);
    return true;
  }

  return false;
}

async function waitForAnyText(page, texts, timeout = 15000) {
  const patterns = texts.map((text) => {
    if (text instanceof RegExp) {
      return { source: text.source, flags: text.flags || "i" };
    }
    return { source: String(text), flags: "i" };
  });

  await page.waitForFunction(
    (patterns) => {
      const bodyText = document.body.innerText || "";
      return patterns.some((pattern) => new RegExp(pattern.source, pattern.flags).test(bodyText));
    },
    patterns,
    { timeout }
  );
}

async function maybeCreateRule(page, suffix) {
  const ruleName = `E2E UI High Temperature ${suffix}`;
  const exactRule = page.locator("tbody tr", { hasText: ruleName });
  if (await exactRule.count()) {
    return ruleName;
  }

  const seededRule = page.locator("tbody tr", { hasText: /High Temperature/i }).filter({ hasText: /active/i });
  if (await seededRule.count()) {
    return null;
  }

  await page.getByLabel(/Name/i).fill(ruleName);
  await page.getByLabel(/Metric/i).selectOption("temperature");
  await page.getByLabel(/Operator/i).selectOption(">");
  await page.getByLabel(/Threshold/i).fill("80");
  await page.getByLabel(/Severity/i).selectOption("CRITICAL");
  await page.getByLabel(/Status/i).selectOption("active");

  const createButton = page.getByRole("button", { name: /create rule/i });
  await expect(createButton).toBeVisible({ timeout: 15000 });
  await createButton.click();

  await page.locator("tbody tr", { hasText: ruleName }).first().waitFor({ state: "visible", timeout: 20000 });
  return ruleName;
}

async function maybeCreateNotificationChannel(page, suffix) {
  const channelName = `E2E UI Webhook ${suffix}`;
  const existingChannel = page.locator("tbody tr", { hasText: channelName });
  if (await existingChannel.count()) {
    return channelName;
  }

  await page.getByLabel(/Name/i).fill(channelName);
  await page.getByLabel(/Type/i).selectOption("WEBHOOK");
  await page.getByLabel(/Target/i).fill("http://host.docker.internal:9999/e2e-ui");
  const enabledCheckbox = page.getByLabel(/Enabled/i).first();
  if (await enabledCheckbox.isVisible() && !(await enabledCheckbox.isChecked())) {
    await enabledCheckbox.check();
  }

  const createButton = page.getByRole("button", { name: /create/i });
  await expect(createButton).toBeVisible({ timeout: 15000 });
  await createButton.click();

  await page.locator("tbody tr", { hasText: channelName }).first().waitFor({ state: "visible", timeout: 20000 });
  return channelName;
}

async function maybeCreateSLAPolicy(page) {
  const existingPolicy = page.locator("tbody tr", { hasText: /CRITICAL/i }).filter({ hasText: /manager@example.com/i });
  if (await existingPolicy.count()) {
    return true;
  }

  await page.getByLabel(/Severity/i).selectOption("CRITICAL");
  await page.getByLabel(/Acknowledge Within/i).fill("1");
  await page.getByLabel(/Resolve Within/i).fill("2");
  await page.getByLabel(/Escalation Target/i).fill("manager@example.com");
  const enabledCheckbox = page.getByLabel(/Enabled/i).first();
  if (await enabledCheckbox.isVisible() && !(await enabledCheckbox.isChecked())) {
    await enabledCheckbox.check();
  }

  const saveButton = page.getByRole("button", { name: /(create|update|save)/i }).filter({ hasText: /create|update|save/i }).first();
  if (await saveButton.count()) {
    await saveButton.click();
  }

  await page.locator("tbody tr", { hasText: /CRITICAL/i }).filter({ hasText: /manager@example.com/i }).first().waitFor({ state: "visible", timeout: 20000 });
  return true;
}

async function handleIncidentLifecycle(page, assetId) {
  const incidentRow = page.locator("tbody tr", { hasText: assetId });
  const row = (await incidentRow.count()) ? incidentRow.first() : page.locator("tbody tr", { hasText: /CRITICAL/i }).first();
  if (!(await row.count())) {
    return;
  }

  const actions = ["Assign", "Acknowledge", "Resolve", "Close"];
  for (const action of actions) {
    const button = row.getByRole("button", { name: new RegExp(`^${action}$`, "i") });
    if (await button.count() && await button.first().isVisible()) {
      await button.first().click();
      await page.waitForLoadState("networkidle");

      if (action === "Assign") {
        await fillIfVisible(page.getByLabel(/Assigned To/i), "operator@example.com");
        await clickIfVisible(page.getByRole("button", { name: /assign incident/i }));
      }

      if (action === "Acknowledge") {
        await fillIfVisible(page.getByLabel(/Comment/i), "E2E acknowledged from UI automation");
        await clickIfVisible(page.getByRole("button", { name: /acknowledge incident/i }));
      }

      if (action === "Resolve") {
        await fillIfVisible(page.getByLabel(/Resolution Note/i), "E2E resolved from UI automation");
        await clickIfVisible(page.getByRole("button", { name: /resolve incident/i }));
      }

      if (action === "Close") {
        await fillIfVisible(page.getByLabel(/Comment/i), "E2E closing incident from UI automation");
        await clickIfVisible(page.getByRole("button", { name: /close incident/i }));
      }

      await page.waitForLoadState("networkidle");
    }
  }
}

module.exports = {
  uniqueSuffix,
  login,
  gotoNav,
  clickIfVisible,
  fillIfVisible,
  waitForAnyText,
  maybeCreateRule,
  maybeCreateNotificationChannel,
  maybeCreateSLAPolicy,
  handleIncidentLifecycle,
};
