const http = require("http");
const { ApiClient } = require("./helpers/apiClient");
const { createWebhookChannel, uniqueSuffix } = require("./helpers/testData");
const { waitFor } = require("./helpers/waitFor");

function startWebhookServer(port) {
  const received = [];
  const server = http.createServer((req, res) => {
    let body = "";
    req.on("data", (chunk) => {
      body += chunk;
    });
    req.on("end", () => {
      received.push(JSON.parse(body || "{}"));
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true }));
    });
  });

  return new Promise((resolve) => {
    server.listen(port, "0.0.0.0", () => {
      resolve({
        received,
        close: () => new Promise((done) => server.close(done))
      });
    });
  });
}

describe("notification API E2E", () => {
  let api;
  let webhook;

  beforeEach(async () => {
    api = new ApiClient();
    await api.login();
  });

  afterEach(async () => {
    if (webhook) {
      await webhook.close();
      webhook = undefined;
    }
  });

  it("sends a test notification to a webhook channel and records SENT history", async () => {
    const port = Number(process.env.E2E_WEBHOOK_PORT || 9100);
    webhook = await startWebhookServer(port);

    const channel = await api.createNotificationChannel(createWebhookChannel(uniqueSuffix(), port));
    await api.sendTestNotification({
      channel_id: channel.id,
      subject: "E2E test notification",
      message: "E2E webhook delivery"
    });

    await waitFor({
      name: "webhook payload",
      timeoutMs: 30000,
      intervalMs: 500,
      action: async () => webhook.received,
      predicate: (items) => items.some((item) => item.event_type === "TEST_NOTIFICATION")
    });

    const history = await waitFor({
      name: "SENT test notification history",
      action: () => api.listNotificationHistory({ event_type: "TEST_NOTIFICATION", status: "SENT" }),
      predicate: (items) => items.some((item) => item.recipient === channel.target)
    });

    expect(history.some((item) => item.recipient === channel.target && item.status === "SENT")).toBe(true);
  });

  it("records FAILED history for an unreachable webhook channel", async () => {
    const channel = await api.createNotificationChannel({
      ...createWebhookChannel(uniqueSuffix(), 9999),
      target: "http://host.docker.internal:9999/webhook"
    });

    await api.sendTestNotification({
      channel_id: channel.id,
      subject: "E2E failed test notification",
      message: "E2E expected failure"
    });

    const history = await waitFor({
      name: "FAILED test notification history",
      action: () => api.listNotificationHistory({ event_type: "TEST_NOTIFICATION", status: "FAILED" }),
      predicate: (items) => items.some((item) => item.recipient === channel.target)
    });

    const failed = history.find((item) => item.recipient === channel.target);
    expect(failed.status).toBe("FAILED");
    expect(failed.error_message).toBeTruthy();
    await expect(api.retryNotification(failed.id)).resolves.toBeTruthy();
  });
});
