const { expect, test } = require("@playwright/test");
const { gotoNav, login, uniqueSuffix } = require("../helpers/uiHelpers");

test("maintenance page creates and completes a task", async ({ page }) => {
  const suffix = uniqueSuffix();
  const title = `E2E UI maintenance ${suffix}`;
  const assetId = `e2e-ui-maint-${suffix}`;

  await login(page);
  await gotoNav(page, "Maintenance");
  await expect(page.getByRole("heading", { name: /^Maintenance$/i })).toBeVisible();

  const form = page.locator(".maintenance-form");
  await form.getByLabel(/Asset ID/i).fill(assetId);
  await form.getByLabel(/^Title$/i).fill(title);
  await form.getByLabel(/^Type$/i).fill("inspection");
  await form.getByLabel(/Priority/i).selectOption("medium");
  await form.getByLabel(/Assigned To/i).fill("operator@example.com");
  await form.getByLabel(/Description/i).fill("Created by Playwright maintenance flow");

  await page.getByRole("button", { name: /create task/i }).click();

  const row = page.locator("tbody tr", { hasText: title }).first();
  await expect(row).toBeVisible({ timeout: 20000 });
  await expect(row).toContainText("scheduled");

  await row.getByRole("button", { name: /^complete$/i }).click();
  await expect(row).toContainText("completed", { timeout: 20000 });
});
