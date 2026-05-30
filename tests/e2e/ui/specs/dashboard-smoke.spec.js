const { expect, test } = require("@playwright/test");

async function login(page) {
  await page.goto("/");
  const loginButton = page.getByRole("button", { name: /^login$/i });
  if (await loginButton.isVisible()) {
    await page.getByPlaceholder("admin@example.com").fill("admin@example.com");
    await page.getByPlaceholder("admin123").fill("admin123");
    await loginButton.click();
  }
}

test("dashboard shows core monitoring widgets", async ({ page }) => {
  await login(page);

  await expect(page.getByText(/Total Assets/i)).toBeVisible();
  await expect(page.getByText(/Total Alerts/i)).toBeVisible();
  await expect(page.getByText(/Open Alerts/i)).toBeVisible();
  await expect(page.getByText(/Telemetry Simulator/i)).toBeVisible();
  const insights = page.locator("section.table-card").filter({ hasText: "Maintenance Insights" });
  await expect(insights.getByRole("heading", { name: /Maintenance Insights/i })).toBeVisible();
  await expect(insights.getByRole("columnheader", { name: /^Asset$/i })).toBeVisible();
  await expect(insights.getByRole("columnheader", { name: /Risk Level/i })).toBeVisible();
  await expect(insights.getByRole("columnheader", { name: /Recommendation/i })).toBeVisible();
  await expect(
    insights.getByText(/No maintenance insights available/i).or(insights.locator("tbody tr").filter({ hasText: /low|medium|high|critical/i }).first())
  ).toBeVisible();
});
