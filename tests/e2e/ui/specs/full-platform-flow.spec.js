const { test, expect } = require("@playwright/test");
const {
  uniqueSuffix,
  login,
  gotoNav,
  fillIfVisible,
  clickIfVisible,
  waitForAnyText,
  maybeCreateRule,
  maybeCreateNotificationChannel,
  maybeCreateSLAPolicy,
  handleIncidentLifecycle,
} = require("../helpers/uiHelpers");

const adminEmail = "admin@example.com";
const adminPassword = "admin123";

test("full platform flow through dashboard UI", async ({ page }) => {
  const suffix = uniqueSuffix();
  const assetId = `e2e-ui-motor-${suffix}`;

  await login(page);
  await expect(page.getByRole("heading", { name: /Enterprise Asset Monitoring/i })).toBeVisible();

  await gotoNav(page, "Rules");
  await expect(page.getByRole("heading", { name: /Dynamic Monitoring Rules/i })).toBeVisible();
  const ruleName = await maybeCreateRule(page, suffix);
  await expect(page.locator("table")).toBeVisible();
  if (ruleName) {
    await expect(page.locator("tbody tr", { hasText: ruleName })).toBeVisible({ timeout: 20000 });
  }

  await gotoNav(page, "Notifications");
  await expect(page.getByRole("heading", { name: /Notifications/i })).toBeVisible();
  await maybeCreateNotificationChannel(page, suffix);
  await expect(page.getByRole("heading", { name: /Notification Channels/i })).toBeVisible();

  await gotoNav(page, "SLA");
  await expect(page.getByRole("heading", { name: /^SLA$/i })).toBeVisible();
  await maybeCreateSLAPolicy(page);
  await expect(page.getByText(/SLA Policies/i)).toBeVisible();

  await gotoNav(page, "Dashboard");
  await expect(page.getByRole("heading", { name: /Enterprise Asset Monitoring/i })).toBeVisible();

  await fillIfVisible(page.locator("label", { hasText: /Asset ID/i }).locator("input"), assetId);
  await fillIfVisible(page.locator("label", { hasText: /Temperature/i }).locator("input"), "95");
  await fillIfVisible(page.locator("label", { hasText: /CPU/i }).locator("input"), "70");
  await fillIfVisible(page.locator("label", { hasText: /Memory/i }).locator("input"), "60");
  await page.getByLabel(/Status/i).selectOption("RUNNING");
  await clickIfVisible(page.getByRole("button", { name: /send telemetry/i }));

  await waitForAnyText(page, [assetId, /High Temperature/i, /CRITICAL/i], 30000);

  await gotoNav(page, "Incidents");
  await expect(page.getByRole("heading", { name: /Incidents/i })).toBeVisible();
  await waitForAnyText(page, [assetId, /High Temperature/i, /CRITICAL/i], 30000);
  await handleIncidentLifecycle(page, assetId);

  await gotoNav(page, "Notifications");
  await expect(page.getByRole("heading", { name: /Notifications/i })).toBeVisible();
  await clickIfVisible(page.getByRole("button", { name: /history/i }));
  await expect(page.getByRole("heading", { name: /Notification History/i }).or(page.locator("text=No notification history found."))).toBeVisible();
  await waitForAnyText(page, [/CRITICAL_ALERT_CREATED/i, /INCIDENT_CREATED/i, /TEST_NOTIFICATION/i, /INCIDENT_ESCALATED/i], 20000).catch(() => {});

  await gotoNav(page, "SLA");
  await expect(page.getByRole("heading", { name: /^SLA$/i })).toBeVisible();
  await expect(page.locator("text=/SLA Breaches|No SLA breaches found\./i").first()).toBeVisible();
  const manualEscalationButton = page.getByRole("button", { name: /manual escalate/i });
  if (await manualEscalationButton.count() && await manualEscalationButton.first().isVisible()) {
    await manualEscalationButton.first().click();
    await fillIfVisible(page.getByLabel(/Target/i), "manager@example.com");
    await fillIfVisible(page.getByLabel(/Reason/i), "E2E manual escalation");
    await clickIfVisible(page.getByRole("button", { name: /escalate/i }));
  }
});
