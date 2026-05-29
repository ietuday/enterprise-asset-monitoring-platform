const axios = require("axios");

const API_BASE_URL = process.env.API_BASE_URL || "http://localhost:4000";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function parseRetryAfter(value) {
  if (!value) return undefined;

  const seconds = Number(value);
  if (!Number.isNaN(seconds)) {
    return Math.max(0, seconds * 1000);
  }

  const dateMs = Date.parse(value);
  if (!Number.isNaN(dateMs)) {
    return Math.max(0, dateMs - Date.now());
  }

  return undefined;
}

class ApiClient {
  constructor() {
    this.baseURL = API_BASE_URL;
    this.token = "";
    this.client = axios.create({
      baseURL: this.baseURL,
      timeout: 15000
    });
  }

  async login(email = "admin@example.com", password = "admin123") {
    const response = await this.request("post", "/api/auth/login", { email, password }, false);
    this.token = response.token;
    return response;
  }

  async get(path, params) {
    return this.request("get", path, undefined, true, { params });
  }

  async post(path, body) {
    return this.request("post", path, body);
  }

  async put(path, body) {
    return this.request("put", path, body);
  }

  async patch(path, body) {
    return this.request("patch", path, body);
  }

  async delete(path) {
    return this.request("delete", path);
  }

  async request(method, path, body, protectedRoute = true, options = {}) {
    const maxAttempts = Number(process.env.E2E_HTTP_MAX_ATTEMPTS || 8);

    for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
      try {
        const response = await this.client.request({
          method,
          url: path,
          data: body,
          headers: protectedRoute && this.token ? { Authorization: `Bearer ${this.token}` } : {},
          ...options
        });
        return response.data;
      } catch (err) {
        const status = err && err.response && err.response.status;

        // API Gateway rate limiting can be hit during E2E polling on a shared local environment.
        // Back off and retry 429s instead of failing/flaking immediately.
        if (status === 429 && attempt < maxAttempts) {
          const retryAfterMs = parseRetryAfter(err.response.headers && err.response.headers["retry-after"]);
          const backoffMs = retryAfterMs || Math.min(1000 * attempt, 5000);
          await sleep(backoffMs);
          continue;
        }

        if (err.response) {
          throw new Error(
            `${method.toUpperCase()} ${this.baseURL}${path} failed with ${err.response.status}: ` +
            JSON.stringify(err.response.data)
          );
        }

        throw new Error(`${method.toUpperCase()} ${this.baseURL}${path} failed: ${err.message}`);
      }
    }

    throw new Error(`${method.toUpperCase()} ${this.baseURL}${path} failed after ${maxAttempts} attempts`);
  }

  createAsset(asset) {
    return this.post("/api/assets", asset);
  }

  listAssets() {
    return this.get("/api/assets");
  }

  createRule(rule) {
    return this.post("/api/rules", rule);
  }

  listRules(status = "") {
    return this.get("/api/rules", status ? { status } : undefined);
  }

  createSLAPolicy(policy) {
    return this.post("/api/sla-policies", policy);
  }

  listSLAPolicies() {
    return this.get("/api/sla-policies");
  }

  createNotificationChannel(channel) {
    return this.post("/api/notification-channels", channel);
  }

  listNotificationChannels() {
    return this.get("/api/notification-channels");
  }

  sendTestNotification(request) {
    return this.post("/api/notifications/test", request);
  }

  listNotificationHistory(filters = {}) {
    return this.get("/api/notifications/history", filters);
  }

  retryNotification(id) {
    return this.post(`/api/notifications/${id}/retry`);
  }

  sendTelemetry(payload) {
    return this.post("/api/telemetry", payload);
  }

  listAlerts() {
    return this.get("/api/alerts");
  }

  listIncidents(filters = {}) {
    return this.get("/api/incidents", filters);
  }

  getIncident(id) {
    return this.get(`/api/incidents/${id}`);
  }

  assignIncident(id, body) {
    return this.put(`/api/incidents/${id}/assign`, body);
  }

  acknowledgeIncident(id, body) {
    return this.put(`/api/incidents/${id}/acknowledge`, body);
  }

  resolveIncident(id, body) {
    return this.put(`/api/incidents/${id}/resolve`, body);
  }

  closeIncident(id, body) {
    return this.put(`/api/incidents/${id}/close`, body);
  }

  listIncidentHistory(id) {
    return this.get(`/api/incidents/${id}/history`);
  }

  getIncidentSLA(id) {
    return this.get(`/api/incidents/${id}/sla`);
  }

  listSLABreaches(filters = {}) {
    return this.get("/api/sla-breaches", filters);
  }

  escalateIncident(id, body) {
    return this.post(`/api/incidents/${id}/escalate`, body);
  }

  listIncidentEscalations(id) {
    return this.get(`/api/incidents/${id}/escalations`);
  }

  createMaintenanceTask(task) {
    return this.post("/api/maintenance/tasks", task);
  }

  listMaintenanceTasks(filters = {}) {
    return this.get("/api/maintenance/tasks", filters);
  }

  completeMaintenanceTask(id, body) {
    return this.post(`/api/maintenance/tasks/${id}/complete`, body);
  }

  listMaintenanceHistory(id) {
    return this.get(`/api/maintenance/history/${id}`);
  }

  listAssetHealth() {
    return this.get("/api/reports/asset-health");
  }
}

module.exports = {
  ApiClient,
  API_BASE_URL
};
