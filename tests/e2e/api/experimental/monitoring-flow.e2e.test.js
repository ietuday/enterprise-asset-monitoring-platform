const http = require("http");
const { ApiClient } = require("./helpers/apiClient");
const { createWebhookChannel } = require("./helpers/testData");
const { createIncidentScenario, waitForNotification, getAssetId, pick } = require("./helpers/flow");

function startWebhookServer(port) {
  const received = [];
  const server = http.createServer((req, res) => {
    if (req.method !== "POST" || req.url !== "/webhook") {
      res.writeHead(404);
      res.end();
      return;
    }

    let body = "";
    req.on("data", (chunk) => {
      body += chunk;
    });
    req.on("end", () => {
      received.push(JSON.parse(body || "{}"));
      res.writeHead(204);
      res.end();
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

describe("monitoring API E2E", () => {
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

  it("validates auth to asset to telemetry to alert to incident to SLA to notification flow", async () => {
    const webhookPort = Number(process.env.E2E_WEBHOOK_PORT || 9100);
    webhook = await startWebhookServer(webhookPort);
    const channel = await api.createNotificationChannel(createWebhookChannel(undefined, webhookPort));

    const { asset, alert, incident } = await createIncidentScenario(api);

    expect(alert.assetId).toBe(asset.id);
    expect(alert.severity).toBe("CRITICAL");
    expect(getAssetId(incident)).toBe(asset.id);
    expect(incident.status).toBe("OPEN");
    expect(incident.severity).toBe("CRITICAL");

    const sla = await api.getIncidentSLA(incident.id);
    expect(String(pick(sla, "incident_id", "incidentId"))).toBe(String(incident.id));
    expect(["ON_TRACK", "ACK_BREACHED", "NO_POLICY"]).toContain(sla.status);

    await waitForNotification(api, "CRITICAL_ALERT_CREATED", (item) => {
      return item.recipient === channel.target && item.message.includes(asset.id);
    });
    await waitForNotification(api, "INCIDENT_CREATED", (item) => {
      return item.recipient === channel.target && item.message.includes(asset.id);
    });
  });
});
