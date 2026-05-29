import axios from "axios";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:4000";

function authHeaders() {
  const token = localStorage.getItem("token");
  return { Authorization: `Bearer ${token}` };
}

function cleanFilters(filters = {}) {
  return Object.fromEntries(Object.entries(filters).filter(([, value]) => value));
}

export async function listMaintenanceTasks(filters = {}) {
  const response = await axios.get(`${API_BASE_URL}/api/maintenance/tasks`, {
    headers: authHeaders(),
    params: cleanFilters(filters),
  });
  return response.data;
}

export async function createMaintenanceTask(task) {
  const response = await axios.post(`${API_BASE_URL}/api/maintenance/tasks`, task, {
    headers: { ...authHeaders(), "Content-Type": "application/json" },
  });
  return response.data;
}

export async function changeMaintenanceStatus(id, payload) {
  const response = await axios.patch(`${API_BASE_URL}/api/maintenance/tasks/${id}/status`, payload, {
    headers: { ...authHeaders(), "Content-Type": "application/json" },
  });
  return response.data;
}

export async function completeMaintenanceTask(id, payload) {
  const response = await axios.post(`${API_BASE_URL}/api/maintenance/tasks/${id}/complete`, payload, {
    headers: { ...authHeaders(), "Content-Type": "application/json" },
  });
  return response.data;
}

export async function cancelMaintenanceTask(id, payload) {
  const response = await axios.post(`${API_BASE_URL}/api/maintenance/tasks/${id}/cancel`, payload, {
    headers: { ...authHeaders(), "Content-Type": "application/json" },
  });
  return response.data;
}

export async function listAssetHealth() {
  const response = await axios.get(`${API_BASE_URL}/api/reports/asset-health`, {
    headers: authHeaders(),
  });
  return response.data;
}
