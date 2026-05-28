const {
  createUniqueAsset,
  createCriticalTemperatureRule,
  createAbnormalTelemetry,
  createCriticalSLAPolicy,
  uniqueSuffix,
} = require("./testData");

const { waitFor } = require("./waitFor");

function errorText(error) {
  return [
    error && error.message,
    error && error.response && JSON.stringify(error.response.data),
    error && error.response && String(error.response.status),
  ]
    .filter(Boolean)
    .join(" ");
}

function isDuplicateError(error) {
  const text = errorText(error).toLowerCase();

  return (
    text.includes("409") ||
    text.includes("already exists") ||
    text.includes("duplicate key") ||
    text.includes("sqlstate 23505") ||
    text.includes("assets_pkey") ||
    text.includes("sla policy already exists")
  );
}

async function ignoreDuplicate(action, label) {
  try {
    return await action();
  } catch (error) {
    if (isDuplicateError(error)) {
      console.warn(`[e2e] ${label} already exists, continuing`);
      return null;
    }

    throw error;
  }
}

function pick(obj, ...keys) {
  if (!obj) return undefined;

  for (const key of keys) {
    if (obj[key] !== undefined && obj[key] !== null) {
      return obj[key];
    }
  }

  return undefined;
}

function parsePayload(payload) {
  if (!payload) return undefined;

  if (typeof payload === "string") {
    try {
      return JSON.parse(payload);
    } catch (_err) {
      return undefined;
    }
  }

  if (typeof payload === "object") {
    return payload;
  }

  return undefined;
}

function getPayload(item) {
  return parsePayload(item && item.payload);
}

function getAssetId(item) {
  return (
    pick(item, "assetId", "asset_id", "assetID", "asset") ||
    pick(getPayload(item), "assetId", "asset_id", "assetID", "asset") ||
    pick(item && item.asset, "id", "assetId", "asset_id", "assetID")
  );
}

function getIncidentId(item) {
  return (
    pick(item, "incidentId", "incident_id", "incidentID", "id") ||
    pick(getPayload(item), "incidentId", "incident_id", "incidentID")
  );
}

function getAlertId(item) {
  return (
    pick(item, "alertId", "alert_id", "alertID", "id") ||
    pick(getPayload(item), "alertId", "alert_id", "alertID")
  );
}

function getRuleName(item) {
  return pick(item, "name", "ruleName", "rule_name", "title");
}

function getSeverity(item) {
  return String(pick(item, "severity") || "").toUpperCase();
}

function getStatus(item) {
  return String(pick(item, "status") || "").toUpperCase();
}

function getEventType(item) {
  return pick(item, "event_type", "eventType");
}

function getCreatedAt(item) {
  return pick(item, "created_at", "createdAt");
}

function createdAfter(item, startedAt, toleranceMs = 5000) {
  const value = getCreatedAt(item);
  if (!value) return true;

  const createdAtMs = new Date(value).getTime();
  if (Number.isNaN(createdAtMs)) return true;

  return createdAtMs >= startedAt.getTime() - toleranceMs;
}

function sortNewestFirst(items) {
  return [...items].sort((a, b) => {
    const aTime = new Date(getCreatedAt(a) || 0).getTime();
    const bTime = new Date(getCreatedAt(b) || 0).getTime();
    return bTime - aTime;
  });
}

function isMatchingAlert(alert, assetId, startedAt) {
  return (
    getAssetId(alert) === assetId &&
    getSeverity(alert) === "CRITICAL" &&
    getStatus(alert) === "OPEN" &&
    createdAfter(alert, startedAt)
  );
}

function isMatchingIncident(incident, assetId, startedAt) {
  const status = getStatus(incident);

  return (
    getAssetId(incident) === assetId &&
    getSeverity(incident) === "CRITICAL" &&
    ["OPEN", "ASSIGNED", "ACKNOWLEDGED"].includes(status) &&
    createdAfter(incident, startedAt)
  );
}

async function createIncidentScenario(api, options = {}) {
  const suffix = uniqueSuffix();

  // createUniqueAsset/createCriticalTemperatureRule expect a suffix string.
  // Passing an object creates IDs like e2e-motor-[object Object], which causes duplicate-key failures.
  const asset = options.asset || createUniqueAsset(suffix);
  const rule = options.rule || createCriticalTemperatureRule(suffix);

  await ignoreDuplicate(() => api.createAsset(asset), `asset ${asset.id}`);
  await ignoreDuplicate(() => api.createRule(rule), `rule ${rule.name}`);

  await ignoreDuplicate(
    () => api.createSLAPolicy(createCriticalSLAPolicy()),
    "CRITICAL SLA policy"
  );

  // Use a timestamp marker so old E2E records in a reused local DB do not confuse matching.
  const startedAt = new Date();
  await api.sendTelemetry(createAbnormalTelemetry(asset.id));

  const alert = await waitFor({
    name: `CRITICAL alert for ${asset.id}`,
    timeoutMs: 90000,
    intervalMs: 1000,
    action: async () => {
      const alerts = await api.listAlerts();
      return sortNewestFirst(alerts).find((item) =>
        isMatchingAlert(item, asset.id, startedAt)
      );
    },
    predicate: Boolean,
  });

  const incident = await waitFor({
    name: `incident for ${asset.id}`,
    timeoutMs: 90000,
    intervalMs: 1000,
    action: async () => {
      const incidents = await api.listIncidents();
      return sortNewestFirst(incidents).find((item) =>
        isMatchingIncident(item, asset.id, startedAt)
      );
    },
    predicate: Boolean,
  });

  return {
    asset,
    rule,
    alert,
    incident,
    alertId: getAlertId(alert),
    incidentId: getIncidentId(incident),
    startedAt,
  };
}

function buildNotificationOptions(eventTypeOrOptions, predicate) {
  if (typeof eventTypeOrOptions === "string") {
    return {
      eventType: eventTypeOrOptions,
      customPredicate: predicate,
    };
  }

  return eventTypeOrOptions || {};
}

async function waitForNotification(api, eventTypeOrOptions = {}, predicate) {
  const {
    eventType,
    assetId,
    alertId,
    incidentId,
    messageIncludes,
    customPredicate,
    startedAt,
    timeoutMs = 90000,
  } = buildNotificationOptions(eventTypeOrOptions, predicate);

  return waitFor({
    name: `${eventType || "matching"} notification`,
    timeoutMs,
    intervalMs: 1000,
    action: async () => {
      const history = await api.listNotificationHistory();

      return sortNewestFirst(history).find((item) => {
        const itemEventType = getEventType(item);
        const itemMessage = item.message || "";
        const payload = getPayload(item);

        const itemAssetId =
          pick(item, "assetId", "asset_id") || pick(payload, "assetId", "asset_id");

        const itemAlertId =
          pick(item, "alertId", "alert_id") || pick(payload, "alertId", "alert_id");

        const itemIncidentId =
          pick(item, "incidentId", "incident_id") || pick(payload, "incidentId", "incident_id");

        if (eventType && itemEventType !== eventType) {
          return false;
        }

        if (startedAt && !createdAfter(item, startedAt)) {
          return false;
        }

        if (assetId && itemAssetId !== assetId && !itemMessage.includes(assetId)) {
          return false;
        }

        if (
          alertId &&
          String(itemAlertId) !== String(alertId) &&
          !itemMessage.includes(String(alertId))
        ) {
          return false;
        }

        if (
          incidentId &&
          String(itemIncidentId) !== String(incidentId) &&
          !itemMessage.includes(`#${incidentId}`) &&
          !itemMessage.includes(String(incidentId))
        ) {
          return false;
        }

        if (messageIncludes && !itemMessage.includes(messageIncludes)) {
          return false;
        }

        if (customPredicate && !customPredicate(item)) {
          return false;
        }

        return true;
      });
    },
    predicate: Boolean,
  });
}

module.exports = {
  createIncidentScenario,
  waitForNotification,
  pick,
  parsePayload,
  getAssetId,
  getIncidentId,
  getAlertId,
  getRuleName,
  getSeverity,
  getStatus,
  getEventType,
  getCreatedAt,
  createdAfter,
  sortNewestFirst,
};
