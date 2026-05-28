const { defineConfig, devices } = require("@playwright/test");

module.exports = defineConfig({
  testDir: "./specs",
  timeout: 90000,
  expect: {
    timeout: 15000,
  },
  outputDir: "test-results",
  retries: process.env.CI ? 1 : 0,
  reporter: [["list"], ["html", { outputFolder: "playwright-report", open: "never" }]],
  use: {
    baseURL: process.env.DASHBOARD_URL || "http://localhost:3000",
    trace: "retain-on-failure",
    browserName: "chromium",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] }
    }
  ]
});
