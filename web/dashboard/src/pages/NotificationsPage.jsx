import { useEffect, useMemo, useState } from "react";
import {
  createNotificationChannel,
  deleteNotificationChannel,
  disableNotificationChannel,
  enableNotificationChannel,
  listNotificationChannels,
  listNotificationHistory,
  retryNotification,
  testNotificationChannel,
  updateNotificationChannel,
} from "../api/notificationsApi";

const emptyChannelForm = {
  name: "",
  type: "WEBHOOK",
  target: "",
  enabled: true,
};

const emptyHistoryFilters = {
  status: "",
  channel_type: "",
  event_type: "",
};

const eventTypes = [
  "CRITICAL_ALERT_CREATED",
  "INCIDENT_CREATED",
  "INCIDENT_ASSIGNED",
  "INCIDENT_ACKNOWLEDGED",
  "INCIDENT_RESOLVED",
  "INCIDENT_CLOSED",
  "TEST_NOTIFICATION",
];

export default function NotificationsPage({ refreshSignal = 0 }) {
  const [activeTab, setActiveTab] = useState("channels");
  const [channels, setChannels] = useState([]);
  const [history, setHistory] = useState([]);
  const [historyFilters, setHistoryFilters] = useState(emptyHistoryFilters);
  const [form, setForm] = useState(emptyChannelForm);
  const [editingID, setEditingID] = useState(null);
  const [loading, setLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState("");
  const [error, setError] = useState("");
  const [formError, setFormError] = useState("");
  const [notice, setNotice] = useState("");

  const user = JSON.parse(localStorage.getItem("user") || "{}");
  const canManage = user?.role === "ADMIN" || user?.role === "OPERATOR";

  const counts = useMemo(() => {
    return history.reduce(
      (current, item) => {
        current[item.status] = (current[item.status] || 0) + 1;
        return current;
      },
      { SENT: 0, FAILED: 0, PENDING: 0 }
    );
  }, [history]);

  async function loadChannels() {
    try {
      setLoading(true);
      setError("");
      setChannels(await listNotificationChannels());
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load notification channels");
    } finally {
      setLoading(false);
    }
  }

  async function loadHistory(nextFilters = historyFilters) {
    try {
      setHistoryLoading(true);
      setError("");
      setHistory(await listNotificationHistory(nextFilters));
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load notification history");
    } finally {
      setHistoryLoading(false);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadChannels();
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadHistory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (refreshSignal > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      loadChannels();
      // eslint-disable-next-line react-hooks/set-state-in-effect
      loadHistory();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshSignal]);

  function handleFormChange(event) {
    const { name, value, checked, type } = event.target;
    setForm((current) => ({
      ...current,
      [name]: type === "checkbox" ? checked : value,
    }));
    setFormError("");
  }

  function validateForm() {
    if (!form.name.trim() || !form.type.trim() || !form.target.trim()) {
      return "Name, type, and target are required.";
    }
    return "";
  }

  async function submitChannel(event) {
    event.preventDefault();
    const validationError = validateForm();
    if (validationError) {
      setFormError(validationError);
      return;
    }

    try {
      setActionLoading("save");
      setError("");
      setNotice("");

      if (editingID) {
        await updateNotificationChannel(editingID, form);
        setNotice("Notification channel updated.");
      } else {
        await createNotificationChannel(form);
        setNotice("Notification channel created.");
      }

      resetForm();
      await loadChannels();
    } catch (err) {
      setFormError(err.response?.data?.error || "Failed to save notification channel");
    } finally {
      setActionLoading("");
    }
  }

  function editChannel(channel) {
    setEditingID(channel.id);
    setForm({
      name: channel.name,
      type: channel.type,
      target: channel.target,
      enabled: channel.enabled,
    });
    setFormError("");
    setNotice("");
  }

  function resetForm() {
    setEditingID(null);
    setForm(emptyChannelForm);
    setFormError("");
  }

  async function runChannelAction(action, channel) {
    try {
      setActionLoading(`${action}-${channel.id}`);
      setError("");
      setNotice("");

      if (action === "enable") {
        await enableNotificationChannel(channel.id);
        setNotice("Notification channel enabled.");
      }
      if (action === "disable") {
        await disableNotificationChannel(channel.id);
        setNotice("Notification channel disabled.");
      }
      if (action === "delete") {
        await deleteNotificationChannel(channel.id);
        setNotice("Notification channel deleted.");
      }
      if (action === "test") {
        await testNotificationChannel(channel.id);
        setNotice("Test notification sent.");
        await loadHistory();
      }

      await loadChannels();
    } catch (err) {
      setError(err.response?.data?.error || "Notification channel action failed");
    } finally {
      setActionLoading("");
    }
  }

  function handleHistoryFilterChange(event) {
    const { name, value } = event.target;
    const nextFilters = {
      ...historyFilters,
      [name]: value,
    };
    setHistoryFilters(nextFilters);
    loadHistory(nextFilters);
  }

  async function retryHistoryItem(item) {
    try {
      setActionLoading(`retry-${item.id}`);
      setError("");
      setNotice("");
      await retryNotification(item.id);
      setNotice("Notification retry requested.");
      await loadHistory();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to retry notification");
    } finally {
      setActionLoading("");
    }
  }

  function formatDate(value) {
    if (!value) return "-";
    return new Date(value).toLocaleString();
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Notifications</h1>
          <p>Manage delivery channels, inspect delivery history, and retry failures.</p>
        </div>

        <button onClick={() => {
          loadChannels();
          loadHistory();
        }}>
          Refresh
        </button>
      </div>

      {error && <div className="error">{error}</div>}
      {notice && <div className="info">{notice}</div>}

      <section className="summary-grid notifications-summary-grid">
        <SummaryCard title="Channels" value={channels.length} />
        <SummaryCard title="Enabled" value={channels.filter((channel) => channel.enabled).length} />
        <SummaryCard title="Sent" value={counts.SENT} />
        <SummaryCard title="Failed" value={counts.FAILED} />
      </section>

      <div className="tabs">
        <button className={activeTab === "channels" ? "active" : "secondary"} onClick={() => setActiveTab("channels")}>
          Channels
        </button>
        <button className={activeTab === "history" ? "active" : "secondary"} onClick={() => setActiveTab("history")}>
          History
        </button>
      </div>

      {activeTab === "channels" ? (
        <>
          {canManage && (
            <section className="actions-card">
              <form className="form-grid notification-form" onSubmit={submitChannel}>
                <h2>{editingID ? `Edit Channel #${editingID}` : "Create Channel"}</h2>

                <label>
                  Name
                  <input name="name" value={form.name} onChange={handleFormChange} required />
                </label>

                <label>
                  Type
                  <select name="type" value={form.type} onChange={handleFormChange} required>
                    <option value="EMAIL">EMAIL</option>
                    <option value="WEBHOOK">WEBHOOK</option>
                  </select>
                </label>

                <label>
                  Target
                  <input
                    name="target"
                    value={form.target}
                    onChange={handleFormChange}
                    placeholder={form.type === "EMAIL" ? "ops@example.com" : "https://example.com/hooks/alerts"}
                    required
                  />
                </label>

                <label className="checkbox-row">
                  <input name="enabled" type="checkbox" checked={form.enabled} onChange={handleFormChange} />
                  Enabled
                </label>

                <div className="form-actions button-row">
                  <button type="submit" disabled={actionLoading === "save"}>
                    {actionLoading === "save" ? "Saving" : editingID ? "Update" : "Create"}
                  </button>
                  {editingID && <button className="secondary" type="button" onClick={resetForm}>Cancel</button>}
                </div>

                {formError && <div className="field-error">{formError}</div>}
              </form>
            </section>
          )}

          <section className="table-card">
            <h2>Notification Channels</h2>
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Name</th>
                  <th>Type</th>
                  <th>Target</th>
                  <th>Enabled</th>
                  <th>Created At</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {loading && channels.length === 0 && (
                  <tr><td colSpan="7">Loading channels.</td></tr>
                )}

                {!loading && channels.map((channel) => (
                  <tr key={channel.id}>
                    <td>{channel.id}</td>
                    <td>{channel.name}</td>
                    <td><span className="badge">{channel.type}</span></td>
                    <td className="wide-cell">{channel.target}</td>
                    <td>
                      <span className={`badge ${channel.enabled ? "status-sent" : "status-failed"}`}>
                        {channel.enabled ? "YES" : "NO"}
                      </span>
                    </td>
                    <td>{formatDate(channel.created_at)}</td>
                    <td>
                      <div className="table-actions">
                        {canManage && (
                          <>
                            <button className="secondary" onClick={() => editChannel(channel)}>Edit</button>
                            <button
                              className="secondary"
                              disabled={actionLoading === `${channel.enabled ? "disable" : "enable"}-${channel.id}`}
                              onClick={() => runChannelAction(channel.enabled ? "disable" : "enable", channel)}
                            >
                              {channel.enabled ? "Disable" : "Enable"}
                            </button>
                            <button
                              disabled={actionLoading === `test-${channel.id}`}
                              onClick={() => runChannelAction("test", channel)}
                            >
                              Test
                            </button>
                            <button
                              className="danger"
                              disabled={actionLoading === `delete-${channel.id}`}
                              onClick={() => runChannelAction("delete", channel)}
                            >
                              Delete
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}

                {!loading && channels.length === 0 && (
                  <tr><td colSpan="7">No notification channels found.</td></tr>
                )}
              </tbody>
            </table>
          </section>
        </>
      ) : (
        <>
          <section className="actions-card">
            <div className="filters-row notification-filters">
              <label>
                Status
                <select name="status" value={historyFilters.status} onChange={handleHistoryFilterChange}>
                  <option value="">All</option>
                  <option value="PENDING">PENDING</option>
                  <option value="SENT">SENT</option>
                  <option value="FAILED">FAILED</option>
                </select>
              </label>

              <label>
                Channel Type
                <select name="channel_type" value={historyFilters.channel_type} onChange={handleHistoryFilterChange}>
                  <option value="">All</option>
                  <option value="EMAIL">EMAIL</option>
                  <option value="WEBHOOK">WEBHOOK</option>
                </select>
              </label>

              <label>
                Event Type
                <select name="event_type" value={historyFilters.event_type} onChange={handleHistoryFilterChange}>
                  <option value="">All</option>
                  {eventTypes.map((eventType) => (
                    <option key={eventType} value={eventType}>{eventType}</option>
                  ))}
                </select>
              </label>
            </div>
          </section>

          <section className="table-card">
            <h2>Notification History</h2>
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Event Type</th>
                  <th>Channel</th>
                  <th>Recipient</th>
                  <th>Status</th>
                  <th>Retry Count</th>
                  <th>Created At</th>
                  <th>Sent At</th>
                  <th>Error</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {historyLoading && history.length === 0 && (
                  <tr><td colSpan="10">Loading notification history.</td></tr>
                )}

                {!historyLoading && history.map((item) => (
                  <tr key={item.id}>
                    <td>{item.id}</td>
                    <td>{item.event_type}</td>
                    <td>{item.channel_name || item.channel_type}</td>
                    <td className="wide-cell">{item.recipient}</td>
                    <td>
                      <span className={`badge status-${item.status.toLowerCase()}`}>
                        {item.status}
                      </span>
                    </td>
                    <td>{item.retry_count}</td>
                    <td>{formatDate(item.created_at)}</td>
                    <td>{formatDate(item.sent_at)}</td>
                    <td className="error-cell">{item.error_message || "-"}</td>
                    <td>
                      {canManage && item.status === "FAILED" && (
                        <button
                          disabled={actionLoading === `retry-${item.id}`}
                          onClick={() => retryHistoryItem(item)}
                        >
                          Retry
                        </button>
                      )}
                    </td>
                  </tr>
                ))}

                {!historyLoading && history.length === 0 && (
                  <tr><td colSpan="10">No notification history found.</td></tr>
                )}
              </tbody>
            </table>
          </section>
        </>
      )}
    </div>
  );
}

function SummaryCard({ title, value }) {
  return (
    <div className="summary-card compact-summary">
      <p>{title}</p>
      <h2>{value}</h2>
    </div>
  );
}
