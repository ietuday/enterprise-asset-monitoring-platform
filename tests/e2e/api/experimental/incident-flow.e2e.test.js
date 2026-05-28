const { ApiClient } = require("./helpers/apiClient");
const { createIncidentScenario, waitForNotification } = require("./helpers/flow");
const { createWebhookChannel, uniqueSuffix } = require("./helpers/testData");

describe("incident lifecycle API E2E", () => {
  let api;

  beforeEach(async () => {
    api = new ApiClient();
    await api.login();
  });

  it("assigns, acknowledges, resolves, and closes an incident", async () => {
    const channel = await api.createNotificationChannel({
      ...createWebhookChannel(uniqueSuffix(), 9999),
      target: `http://host.docker.internal:9999/e2e-${uniqueSuffix()}`
    });
    const { incident } = await createIncidentScenario(api);

    await api.assignIncident(incident.id, {
      assigned_to: "operator@example.com",
      actor: "admin@example.com",
      comment: "E2E assignment"
    });

    await api.acknowledgeIncident(incident.id, {
      actor: "operator@example.com",
      comment: "E2E acknowledge"
    });

    await api.resolveIncident(incident.id, {
      actor: "operator@example.com",
      resolution_note: "E2E resolved after validation"
    });

    const closed = await api.closeIncident(incident.id, {
      actor: "admin@example.com",
      comment: "E2E close"
    });

    expect(closed.status).toBe("CLOSED");

    const history = await api.listIncidentHistory(incident.id);
    const actions = history.map((item) => item.action);
    expect(actions).toEqual(expect.arrayContaining([
      "ASSIGNED",
      "ACKNOWLEDGED",
      "RESOLVED",
      "CLOSED"
    ]));

    await waitForNotification(api, "INCIDENT_ASSIGNED", (item) => item.recipient === channel.target && item.message.includes(`#${incident.id}`));
    await waitForNotification(api, "INCIDENT_ACKNOWLEDGED", (item) => item.recipient === channel.target && item.message.includes(`#${incident.id}`));
    await waitForNotification(api, "INCIDENT_RESOLVED", (item) => item.recipient === channel.target && item.message.includes(`#${incident.id}`));
    await waitForNotification(api, "INCIDENT_CLOSED", (item) => item.recipient === channel.target && item.message.includes(`#${incident.id}`));
  });
});
