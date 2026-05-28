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

test("dashboard navigation reaches the main product pages", async ({ page }) => {
  await login(page);

  await page.getByRole("button", { name: "Dashboard" }).click();
  await expect(page.getByText(/Telemetry Simulator/i)).toBeVisible();

  await page.getByRole("button", { name: "Rules" }).click();
  await expect(page.getByText(/Dynamic Monitoring Rules/i)).toBeVisible();

  await page.getByRole("button", { name: "Incidents" }).click();
  await expect(page.getByRole("heading", { name: /Incidents/i })).toBeVisible();

  await page.getByRole("button", { name: "Notifications" }).click();
  await expect(page.getByRole("heading", { name: /Notifications/i })).toBeVisible();

  await page.getByRole("button", { name: "SLA" }).click();
  await expect(page.locator("h1", { hasText: /^SLA$/i })).toBeVisible();
});
