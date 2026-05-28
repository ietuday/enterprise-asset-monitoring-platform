const { ApiClient } = require("./helpers/apiClient");
const { createIncidentScenario, waitForNotification, pick } = require("./helpers/flow");
const { createWebhookChannel, uniqueSuffix } = require("./helpers/testData");

describe("SLA and escalation API E2E", () => {
  let api;

  beforeEach(async () => {
    api = new ApiClient();
    await api.login();
  });

  it("tracks incident SLA and supports manual escalation", async () => {
    const channel = await api.createNotificationChannel({
      ...createWebhookChannel(uniqueSuffix(), 9999),
      target: `http://host.docker.internal:9999/e2e-${uniqueSuffix()}`
    });
    const { incident } = await createIncidentScenario(api);

    const sla = await api.getIncidentSLA(incident.id);
    expect(String(pick(sla, "incident_id", "incidentId"))).toBe(String(incident.id));
    expect(sla.severity).toBe("CRITICAL");
    expect(["ON_TRACK", "ACK_BREACHED", "NO_POLICY"]).toContain(sla.status);

    const escalation = await api.escalateIncident(incident.id, {
      reason: "E2E manual escalation",
      target: "manager@example.com",
      actor: "admin@example.com"
    });

    expect(escalation.action).toBe("INCIDENT_ESCALATED");

    const escalations = await api.listIncidentEscalations(incident.id);
    expect(escalations.some((item) => item.action === "INCIDENT_ESCALATED")).toBe(true);

    await waitForNotification(api, "INCIDENT_ESCALATED", (item) => {
      return item.recipient === channel.target && item.message.includes(`#${incident.id}`);
    });
  });
});
