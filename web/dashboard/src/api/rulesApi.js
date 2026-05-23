import axios from "axios";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:4000";

function authHeaders() {
  const token = localStorage.getItem("token");

  return {
    Authorization: `Bearer ${token}`,
  };
}

export async function getRules(status = "") {
  const response = await axios.get(`${API_BASE_URL}/api/rules`, {
    headers: authHeaders(),
    params: status ? { status } : {},
  });

  return response.data;
}

export async function createRule(rule) {
  const response = await axios.post(`${API_BASE_URL}/api/rules`, rule, {
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
  });

  return response.data;
}

export async function updateRule(id, rule) {
  const response = await axios.put(`${API_BASE_URL}/api/rules/${id}`, rule, {
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
  });

  return response.data;
}

export async function deleteRule(id) {
  const response = await axios.delete(`${API_BASE_URL}/api/rules/${id}`, {
    headers: authHeaders(),
  });

  return response.data;
}

export async function activateRule(id) {
  const response = await axios.patch(
    `${API_BASE_URL}/api/rules/${id}/activate`,
    {},
    {
      headers: authHeaders(),
    }
  );

  return response.data;
}

export async function disableRule(id) {
  const response = await axios.patch(
    `${API_BASE_URL}/api/rules/${id}/disable`,
    {},
    {
      headers: authHeaders(),
    }
  );

  return response.data;
}

export async function archiveRule(id) {
  const response = await axios.patch(
    `${API_BASE_URL}/api/rules/${id}/archive`,
    {},
    {
      headers: authHeaders(),
    }
  );

  return response.data;
}

export async function getRuleVersions(id) {
  const response = await axios.get(`${API_BASE_URL}/api/rules/${id}/versions`, {
    headers: authHeaders(),
  });

  return response.data;
}

export async function rollbackRule(id, version) {
  const response = await axios.post(
    `${API_BASE_URL}/api/rules/${id}/rollback/${version}`,
    {},
    {
      headers: authHeaders(),
    }
  );

  return response.data;
}

export async function getRuleHistory() {
  const response = await axios.get(`${API_BASE_URL}/api/rules/history`, {
    headers: authHeaders(),
  });

  return response.data;
}

export async function getRuleHistoryById(id) {
  const response = await axios.get(`${API_BASE_URL}/api/rules/${id}/history`, {
    headers: authHeaders(),
  });

  return response.data;
}