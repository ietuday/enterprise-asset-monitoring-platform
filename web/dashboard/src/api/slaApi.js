import axios from "axios";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:4000";

function authHeaders() {
  const token = localStorage.getItem("token");
  return { Authorization: `Bearer ${token}` };
}

export async function listSLAPolicies() {
  const response = await axios.get(`${API_BASE_URL}/api/sla-policies`, {
    headers: authHeaders(),
  });
  return response.data;
}

export async function createSLAPolicy(policy) {
  const response = await axios.post(`${API_BASE_URL}/api/sla-policies`, policy, {
    headers: { ...authHeaders(), "Content-Type": "application/json" },
  });
  return response.data;
}

export async function updateSLAPolicy(id, policy) {
  const response = await axios.put(`${API_BASE_URL}/api/sla-policies/${id}`, policy, {
    headers: { ...authHeaders(), "Content-Type": "application/json" },
  });
  return response.data;
}

export async function deleteSLAPolicy(id) {
  await axios.delete(`${API_BASE_URL}/api/sla-policies/${id}`, {
    headers: authHeaders(),
  });
}

export async function getIncidentSLA(id) {
  const response = await axios.get(`${API_BASE_URL}/api/incidents/${id}/sla`, {
    headers: authHeaders(),
  });
  return response.data;
}

export async function listSLABreaches(filters = {}) {
  const params = Object.fromEntries(Object.entries(filters).filter(([, value]) => value));
  const response = await axios.get(`${API_BASE_URL}/api/sla-breaches`, {
    headers: authHeaders(),
    params,
  });
  return response.data;
}

export async function escalateIncident(id, payload) {
  const response = await axios.post(`${API_BASE_URL}/api/incidents/${id}/escalate`, payload, {
    headers: { ...authHeaders(), "Content-Type": "application/json" },
  });
  return response.data;
}

export async function listIncidentEscalations(id) {
  const response = await axios.get(`${API_BASE_URL}/api/incidents/${id}/escalations`, {
    headers: authHeaders(),
  });
  return response.data;
}
