import { useEffect, useMemo, useState } from "react";
import {
  acknowledgeIncident,
  assignIncident,
  closeIncident,
  getIncidentHistory,
  listIncidents,
  resolveIncident,
} from "../api/incidentsApi";

const emptyFilters = {
  status: "",
  severity: "",
  assigned_to: "",
};

const actionLabels = {
  assign: "Assign",
  acknowledge: "Acknowledge",
  resolve: "Resolve",
  close: "Close",
};

const validActionsByStatus = {
  OPEN: ["assign", "acknowledge", "close"],
  ASSIGNED: ["acknowledge", "close"],
  ACKNOWLEDGED: ["resolve", "close"],
  RESOLVED: ["close"],
  CLOSED: [],
};

export default function IncidentsPage({ refreshSignal = 0 }) {
  const [incidents, setIncidents] = useState([]);
  const [filters, setFilters] = useState(emptyFilters);
  const [selectedIncident, setSelectedIncident] = useState(null);
  const [actionMode, setActionMode] = useState("");
  const [assignedTo, setAssignedTo] = useState("");
  const [comment, setComment] = useState("");
  const [resolutionNote, setResolutionNote] = useState("");
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [error, setError] = useState("");
  const [formError, setFormError] = useState("");

  const user = JSON.parse(localStorage.getItem("user") || "{}");
  const canUpdate = user?.role === "ADMIN" || user?.role === "OPERATOR";
  const actor = user?.email || user?.name || "dashboard-user";

  const counts = useMemo(() => {
    return incidents.reduce(
      (current, incident) => {
        current[incident.status] = (current[incident.status] || 0) + 1;
        if (incident.severity === "CRITICAL") {
          current.CRITICAL += 1;
        }
        return current;
      },
      {
        OPEN: 0,
        ASSIGNED: 0,
        ACKNOWLEDGED: 0,
        RESOLVED: 0,
        CLOSED: 0,
        CRITICAL: 0,
      }
    );
  }, [incidents]);

  async function loadIncidents(nextFilters = filters) {
    try {
      setLoading(true);
      setError("");
      const data = await listIncidents(nextFilters);
      setIncidents(data);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load incidents");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadIncidents();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (refreshSignal > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      loadIncidents();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshSignal]);

  function handleFilterChange(event) {
    const { name, value } = event.target;
    const nextFilters = {
      ...filters,
      [name]: value,
    };

    setFilters(nextFilters);
    loadIncidents(nextFilters);
  }

  function openAction(incident, mode) {
    setSelectedIncident(incident);
    setActionMode(mode);
    setAssignedTo(incident.assigned_to || "");
    setComment("");
    setResolutionNote("");
    setHistory([]);
    setFormError("");
  }

  async function openHistory(incident) {
    try {
      setHistoryLoading(true);
      setError("");
      setFormError("");
      setSelectedIncident(incident);
      setActionMode("history");
      setHistory(await getIncidentHistory(incident.id));
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load incident history");
    } finally {
      setHistoryLoading(false);
    }
  }

  function closePanel() {
    setSelectedIncident(null);
    setActionMode("");
    setHistory([]);
    setFormError("");
  }

  async function submitAction(event) {
    event.preventDefault();
    if (!selectedIncident) return;

    const trimmedAssignedTo = assignedTo.trim();
    const trimmedResolutionNote = resolutionNote.trim();
    const trimmedComment = comment.trim();

    if (actionMode === "assign" && !trimmedAssignedTo) {
      setFormError("Assigned To is required.");
      return;
    }

    if (actionMode === "resolve" && !trimmedResolutionNote) {
      setFormError("Resolution note is required.");
      return;
    }

    try {
      setActionLoading(true);
      setError("");
      setFormError("");

      if (actionMode === "assign") {
        await assignIncident(selectedIncident.id, {
          assigned_to: trimmedAssignedTo,
          actor,
          comment: trimmedComment,
        });
      }

      if (actionMode === "acknowledge") {
        await acknowledgeIncident(selectedIncident.id, {
          actor,
          comment: trimmedComment,
        });
      }

      if (actionMode === "resolve") {
        await resolveIncident(selectedIncident.id, {
          actor,
          resolution_note: trimmedResolutionNote,
        });
      }

      if (actionMode === "close") {
        await closeIncident(selectedIncident.id, {
          actor,
          comment: trimmedComment,
        });
      }

      closePanel();
      await loadIncidents();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to update incident");
    } finally {
      setActionLoading(false);
    }
  }

  function formatDate(value) {
    if (!value) return "-";
    return new Date(value).toLocaleString();
  }

  function availableActions(incident) {
    const lifecycleActions = canUpdate
      ? validActionsByStatus[incident.status] || []
      : [];

    return [...lifecycleActions, "history"];
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Incidents</h1>
          <p>Track critical alert response, ownership, resolution, and audit history.</p>
        </div>

        <button onClick={() => loadIncidents()} disabled={loading}>
          {loading ? "Refreshing" : "Refresh"}
        </button>
      </div>

      {error && <div className="error">{error}</div>}

      <section className="summary-grid incidents-summary-grid">
        <SummaryCard title="Open incidents" value={counts.OPEN} />
        <SummaryCard title="Assigned incidents" value={counts.ASSIGNED} />
        <SummaryCard title="Acknowledged incidents" value={counts.ACKNOWLEDGED} />
        <SummaryCard title="Resolved incidents" value={counts.RESOLVED} />
        <SummaryCard title="Closed incidents" value={counts.CLOSED} />
        <SummaryCard title="Critical incidents" value={counts.CRITICAL} />
      </section>

      <section className="actions-card">
        <div className="filters-row">
          <label>
            Status
            <select name="status" value={filters.status} onChange={handleFilterChange}>
              <option value="">All</option>
              <option value="OPEN">OPEN</option>
              <option value="ASSIGNED">ASSIGNED</option>
              <option value="ACKNOWLEDGED">ACKNOWLEDGED</option>
              <option value="RESOLVED">RESOLVED</option>
              <option value="CLOSED">CLOSED</option>
            </select>
          </label>

          <label>
            Severity
            <select name="severity" value={filters.severity} onChange={handleFilterChange}>
              <option value="">All</option>
              <option value="CRITICAL">CRITICAL</option>
              <option value="HIGH">HIGH</option>
              <option value="MEDIUM">MEDIUM</option>
              <option value="LOW">LOW</option>
            </select>
          </label>

          <label>
            Assigned To
            <input
              name="assigned_to"
              value={filters.assigned_to}
              onChange={handleFilterChange}
              placeholder="operator@example.com"
            />
          </label>
        </div>
      </section>

      <section className="table-card">
        <h2>Incident Queue</h2>

        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Title</th>
              <th>Asset</th>
              <th>Severity</th>
              <th>Status</th>
              <th>Assigned To</th>
              <th>Created At</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading && incidents.length === 0 && (
              <tr>
                <td colSpan="8">Loading incidents.</td>
              </tr>
            )}

            {!loading && incidents.map((incident) => (
              <tr key={incident.id}>
                <td>{incident.id}</td>
                <td>{incident.title}</td>
                <td>{incident.asset_id}</td>
                <td>
                  <span className={`badge severity-${incident.severity.toLowerCase()}`}>
                    {incident.severity}
                  </span>
                </td>
                <td>
                  <span className={`badge status-${incident.status.toLowerCase()}`}>
                    {incident.status}
                  </span>
                </td>
                <td>{incident.assigned_to || "-"}</td>
                <td>{formatDate(incident.created_at)}</td>
                <td>
                  <div className="table-actions">
                    {availableActions(incident).map((action) => (
                      <button
                        className={action === "history" ? "" : "secondary"}
                        key={action}
                        onClick={() => (
                          action === "history"
                            ? openHistory(incident)
                            : openAction(incident, action)
                        )}
                      >
                        {action === "history" ? "History" : actionLabels[action]}
                      </button>
                    ))}
                  </div>
                </td>
              </tr>
            ))}

            {!loading && incidents.length === 0 && (
              <tr>
                <td colSpan="8">No incidents found.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>

      {selectedIncident && actionMode !== "history" && (
        <section className="card incident-panel">
          <div className="panel-header">
            <h2>{actionLabels[actionMode]} Incident #{selectedIncident.id}</h2>
            <button className="secondary" onClick={closePanel}>Close</button>
          </div>

          <form onSubmit={submitAction}>
            {actionMode === "assign" && (
              <label>
                Assigned To
                <input
                  value={assignedTo}
                  onChange={(event) => {
                    setAssignedTo(event.target.value);
                    setFormError("");
                  }}
                  required
                />
              </label>
            )}

            {actionMode === "resolve" ? (
              <label>
                Resolution Note
                <textarea
                  value={resolutionNote}
                  onChange={(event) => {
                    setResolutionNote(event.target.value);
                    setFormError("");
                  }}
                  required
                />
              </label>
            ) : (
              <label>
                Comment
                <textarea
                  value={comment}
                  onChange={(event) => setComment(event.target.value)}
                />
              </label>
            )}

            {formError && <div className="field-error">{formError}</div>}

            <button type="submit" disabled={actionLoading}>
              {actionLoading ? "Saving" : `${actionLabels[actionMode]} Incident`}
            </button>
          </form>
        </section>
      )}

      {selectedIncident && actionMode === "history" && (
        <section className="card incident-panel">
          <div className="panel-header">
            <h2>Incident #{selectedIncident.id} History</h2>
            <button className="secondary" onClick={closePanel}>Close</button>
          </div>

          <div className="timeline">
            {historyLoading && <p className="empty-state">Loading history.</p>}

            {!historyLoading && history.map((item) => (
              <div className="timeline-item" key={item.id}>
                <span className={`badge action-${item.action.toLowerCase()}`}>
                  {item.action}
                </span>
                <div className="timeline-status">
                  <strong>{item.old_status || "None"}</strong>
                  <span>to</span>
                  <strong>{item.new_status}</strong>
                </div>
                <p>{item.comment || "No comment provided."}</p>
                <small>Actor: {item.actor || "system"} · {formatDate(item.created_at)}</small>
              </div>
            ))}

            {!historyLoading && history.length === 0 && (
              <p className="empty-state">No incident history found.</p>
            )}
          </div>
        </section>
      )}
    </div>
  );
}

function SummaryCard({ title, value }) {
  return (
    <div className="summary-card">
      <p>{title}</p>
      <h2>{value}</h2>
    </div>
  );
}
