import axios from "axios";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:4000";

function authHeaders() {
  const token = localStorage.getItem("token");

  return {
    Authorization: `Bearer ${token}`,
  };
}

export async function listIncidents(filters = {}) {
  const params = Object.fromEntries(
    Object.entries(filters).filter(([, value]) => value)
  );

  const response = await axios.get(`${API_BASE_URL}/api/incidents`, {
    headers: authHeaders(),
    params,
  });

  return response.data;
}

export async function getIncident(id) {
  const response = await axios.get(`${API_BASE_URL}/api/incidents/${id}`, {
    headers: authHeaders(),
  });

  return response.data;
}

export async function createIncident(incident) {
  const response = await axios.post(`${API_BASE_URL}/api/incidents`, incident, {
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
  });

  return response.data;
}

export async function assignIncident(id, payload) {
  const response = await axios.put(
    `${API_BASE_URL}/api/incidents/${id}/assign`,
    payload,
    {
      headers: {
        ...authHeaders(),
        "Content-Type": "application/json",
      },
    }
  );

  return response.data;
}

export async function acknowledgeIncident(id, payload) {
  const response = await axios.put(
    `${API_BASE_URL}/api/incidents/${id}/acknowledge`,
    payload,
    {
      headers: {
        ...authHeaders(),
        "Content-Type": "application/json",
      },
    }
  );

  return response.data;
}

export async function resolveIncident(id, payload) {
  const response = await axios.put(
    `${API_BASE_URL}/api/incidents/${id}/resolve`,
    payload,
    {
      headers: {
        ...authHeaders(),
        "Content-Type": "application/json",
      },
    }
  );

  return response.data;
}

export async function closeIncident(id, payload) {
  const response = await axios.put(
    `${API_BASE_URL}/api/incidents/${id}/close`,
    payload,
    {
      headers: {
        ...authHeaders(),
        "Content-Type": "application/json",
      },
    }
  );

  return response.data;
}

export async function getIncidentHistory(id) {
  const response = await axios.get(`${API_BASE_URL}/api/incidents/${id}/history`, {
    headers: authHeaders(),
  });

  return response.data;
}
