const { expect, test } = require("@playwright/test");

async function loginIfNeeded(page) {
  await page.goto("/");

  const loginButton = page.getByRole("button", { name: /^login$/i });
  if (await loginButton.isVisible()) {
    await page.getByPlaceholder("admin@example.com").fill("admin@example.com");
    await page.getByPlaceholder("admin123").fill("admin123");
    await loginButton.click();
  }
}

test("admin can log in to the dashboard", async ({ page }) => {
  await loginIfNeeded(page);

  await expect(page.getByRole("heading", { name: /Enterprise Asset Monitoring/i })).toBeVisible();
  await expect(page.getByText(/ADMIN/i)).toBeVisible();
});
