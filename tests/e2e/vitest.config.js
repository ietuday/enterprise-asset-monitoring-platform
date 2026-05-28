module.exports = {
  test: {
    environment: "node",

    // Enables describe, it, expect, beforeEach, afterEach globally.
    globals: true,

    include: [
      "api/auth.e2e.test.js",
      "api/monitoring-flow.e2e.test.js",
      "api/incident-flow.e2e.test.js",
      "api/notification-flow.e2e.test.js",
      "api/sla-flow.e2e.test.js",
    ],

    // E2E tests hit shared Docker services and API Gateway rate limits.
    // Run sequentially to avoid noisy 429 failures.
    fileParallelism: false,
    maxConcurrency: 1,

    testTimeout: 180000,
    hookTimeout: 60000,
    teardownTimeout: 30000,
  },
};