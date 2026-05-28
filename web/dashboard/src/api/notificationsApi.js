import axios from "axios";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:4000";

function authHeaders() {
  const token = localStorage.getItem("token");

  return {
    Authorization: `Bearer ${token}`,
  };
}

export async function listNotificationChannels() {
  const response = await axios.get(`${API_BASE_URL}/api/notification-channels`, {
    headers: authHeaders(),
  });

  return response.data;
}

export async function createNotificationChannel(channel) {
  const response = await axios.post(`${API_BASE_URL}/api/notification-channels`, channel, {
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
  });

  return response.data;
}

export async function updateNotificationChannel(id, channel) {
  const response = await axios.put(`${API_BASE_URL}/api/notification-channels/${id}`, channel, {
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
  });

  return response.data;
}

export async function deleteNotificationChannel(id) {
  await axios.delete(`${API_BASE_URL}/api/notification-channels/${id}`, {
    headers: authHeaders(),
  });
}

export async function enableNotificationChannel(id) {
  const response = await axios.patch(`${API_BASE_URL}/api/notification-channels/${id}/enable`, null, {
    headers: authHeaders(),
  });

  return response.data;
}

export async function disableNotificationChannel(id) {
  const response = await axios.patch(`${API_BASE_URL}/api/notification-channels/${id}/disable`, null, {
    headers: authHeaders(),
  });

  return response.data;
}

export async function testNotificationChannel(channelID) {
  const response = await axios.post(
    `${API_BASE_URL}/api/notifications/test`,
    {
      channel_id: channelID,
      subject: "Test notification",
      message: "This is a test notification from Enterprise Asset Monitoring Platform.",
    },
    {
      headers: {
        ...authHeaders(),
        "Content-Type": "application/json",
      },
    }
  );

  return response.data;
}

export async function listNotificationHistory(filters = {}) {
  const params = Object.fromEntries(
    Object.entries(filters).filter(([, value]) => value)
  );

  const response = await axios.get(`${API_BASE_URL}/api/notifications/history`, {
    headers: authHeaders(),
    params,
  });

  return response.data;
}

export async function retryNotification(id) {
  const response = await axios.post(`${API_BASE_URL}/api/notifications/${id}/retry`, null, {
    headers: authHeaders(),
  });

  return response.data;
}
