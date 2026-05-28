import { useEffect, useMemo, useState } from "react";
import {
  createSLAPolicy,
  deleteSLAPolicy,
  escalateIncident,
  listIncidentEscalations,
  listSLABreaches,
  listSLAPolicies,
  updateSLAPolicy,
} from "../api/slaApi";

const severities = ["CRITICAL", "HIGH", "MEDIUM", "LOW"];
const statuses = ["ACK_BREACHED", "RESOLUTION_BREACHED", "ESCALATED"];

const emptyPolicyForm = {
  severity: "CRITICAL",
  acknowledge_within_minutes: 5,
  resolve_within_minutes: 30,
  escalation_target: "",
  enabled: true,
};

const emptyFilters = {
  status: "",
  severity: "",
  incident_id: "",
};

export default function SlaPage({ refreshSignal = 0 }) {
  const [policies, setPolicies] = useState([]);
  const [breaches, setBreaches] = useState([]);
  const [filters, setFilters] = useState(emptyFilters);
  const [form, setForm] = useState(emptyPolicyForm);
  const [editingID, setEditingID] = useState(null);
  const [selectedIncidentID, setSelectedIncidentID] = useState("");
  const [escalations, setEscalations] = useState([]);
  const [manualForm, setManualForm] = useState({ reason: "", target: "", actor: "" });
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState("");
  const [error, setError] = useState("");
  const [formError, setFormError] = useState("");
  const [notice, setNotice] = useState("");

  const user = JSON.parse(localStorage.getItem("user") || "{}");
  const canManagePolicies = user?.role === "ADMIN";
  const canEscalate = user?.role === "ADMIN" || user?.role === "OPERATOR";
  const actor = user?.email || user?.name || "dashboard-user";

  const counts = useMemo(() => breaches.reduce(
    (current, item) => {
      current[item.status] = (current[item.status] || 0) + 1;
      return current;
    },
    { ACK_BREACHED: 0, RESOLUTION_BREACHED: 0, ESCALATED: 0 }
  ), [breaches]);

  async function loadAll(nextFilters = filters) {
    try {
      setLoading(true);
      setError("");
      const [policyData, breachData] = await Promise.all([
        listSLAPolicies(),
        listSLABreaches(nextFilters),
      ]);
      setPolicies(policyData);
      setBreaches(breachData);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load SLA data");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (refreshSignal > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      loadAll();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshSignal]);

  function handlePolicyChange(event) {
    const { name, value, type, checked } = event.target;
    setForm((current) => ({
      ...current,
      [name]: type === "checkbox" ? checked : value,
    }));
    setFormError("");
  }

  function validatePolicy() {
    const ack = Number(form.acknowledge_within_minutes);
    const resolve = Number(form.resolve_within_minutes);
    if (!form.severity || !form.escalation_target.trim()) return "Severity and escalation target are required.";
    if (ack <= 0) return "Acknowledge minutes must be greater than 0.";
    if (resolve <= 0) return "Resolve minutes must be greater than 0.";
    if (resolve < ack) return "Resolve minutes must be greater than or equal to acknowledge minutes.";
    return "";
  }

  async function submitPolicy(event) {
    event.preventDefault();
    const validation = validatePolicy();
    if (validation) {
      setFormError(validation);
      return;
    }

    const payload = {
      ...form,
      acknowledge_within_minutes: Number(form.acknowledge_within_minutes),
      resolve_within_minutes: Number(form.resolve_within_minutes),
    };

    try {
      setActionLoading("policy");
      setError("");
      setNotice("");
      if (editingID) {
        await updateSLAPolicy(editingID, payload);
        setNotice("SLA policy updated.");
      } else {
        await createSLAPolicy(payload);
        setNotice("SLA policy created.");
      }
      resetPolicyForm();
      await loadAll();
    } catch (err) {
      setFormError(err.response?.data?.error || "Failed to save SLA policy");
    } finally {
      setActionLoading("");
    }
  }

  function editPolicy(policy) {
    setEditingID(policy.id);
    setForm({
      severity: policy.severity,
      acknowledge_within_minutes: policy.acknowledge_within_minutes,
      resolve_within_minutes: policy.resolve_within_minutes,
      escalation_target: policy.escalation_target,
      enabled: policy.enabled,
    });
    setFormError("");
  }

  function resetPolicyForm() {
    setEditingID(null);
    setForm(emptyPolicyForm);
    setFormError("");
  }

  async function removePolicy(policy) {
    try {
      setActionLoading(`delete-${policy.id}`);
      setError("");
      setNotice("");
      await deleteSLAPolicy(policy.id);
      setNotice("SLA policy deleted.");
      await loadAll();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to delete SLA policy");
    } finally {
      setActionLoading("");
    }
  }

  function handleFilterChange(event) {
    const { name, value } = event.target;
    const nextFilters = { ...filters, [name]: value };
    setFilters(nextFilters);
    loadAll(nextFilters);
  }

  async function viewEscalations(incidentID) {
    try {
      setActionLoading(`history-${incidentID}`);
      setError("");
      setSelectedIncidentID(String(incidentID));
      setEscalations(await listIncidentEscalations(incidentID));
      setManualForm({ reason: "", target: "", actor });
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load escalation history");
    } finally {
      setActionLoading("");
    }
  }

  async function submitManualEscalation(event) {
    event.preventDefault();
    if (!selectedIncidentID) return;
    if (!manualForm.target.trim()) {
      setError("Escalation target is required.");
      return;
    }

    try {
      setActionLoading("manual-escalation");
      setError("");
      setNotice("");
      await escalateIncident(selectedIncidentID, {
        reason: manualForm.reason.trim() || "Manual escalation from dashboard",
        target: manualForm.target.trim(),
        actor: manualForm.actor.trim() || actor,
      });
      setNotice("Incident escalated.");
      await viewEscalations(selectedIncidentID);
      await loadAll();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to escalate incident");
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
          <h1>SLA</h1>
          <p>Manage incident deadlines, breaches, and escalation history.</p>
        </div>
        <button onClick={() => loadAll()} disabled={loading}>{loading ? "Refreshing" : "Refresh"}</button>
      </div>

      {error && <div className="error">{error}</div>}
      {notice && <div className="info">{notice}</div>}

      <section className="summary-grid sla-summary-grid">
        <SummaryCard title="Policies" value={policies.length} />
        <SummaryCard title="Ack Breaches" value={counts.ACK_BREACHED} />
        <SummaryCard title="Resolution Breaches" value={counts.RESOLUTION_BREACHED} />
        <SummaryCard title="Escalated" value={counts.ESCALATED} />
      </section>

      {canManagePolicies && (
        <section className="actions-card">
          <form className="form-grid sla-policy-form" onSubmit={submitPolicy}>
            <h2>{editingID ? `Edit SLA Policy #${editingID}` : "Create SLA Policy"}</h2>
            <label>
              Severity
              <select name="severity" value={form.severity} onChange={handlePolicyChange}>
                {severities.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
              </select>
            </label>
            <label>
              Acknowledge Within
              <input name="acknowledge_within_minutes" type="number" min="1" value={form.acknowledge_within_minutes} onChange={handlePolicyChange} />
            </label>
            <label>
              Resolve Within
              <input name="resolve_within_minutes" type="number" min="1" value={form.resolve_within_minutes} onChange={handlePolicyChange} />
            </label>
            <label>
              Escalation Target
              <input name="escalation_target" value={form.escalation_target} onChange={handlePolicyChange} placeholder="manager@example.com" />
            </label>
            <label className="checkbox-row">
              <input name="enabled" type="checkbox" checked={form.enabled} onChange={handlePolicyChange} />
              Enabled
            </label>
            <div className="form-actions button-row">
              <button type="submit" disabled={actionLoading === "policy"}>{actionLoading === "policy" ? "Saving" : editingID ? "Update" : "Create"}</button>
              {editingID && <button className="secondary" type="button" onClick={resetPolicyForm}>Cancel</button>}
            </div>
            {formError && <div className="field-error">{formError}</div>}
          </form>
        </section>
      )}

      <section className="table-card">
        <h2>SLA Policies</h2>
        <table>
          <thead>
            <tr>
              <th>Severity</th>
              <th>Acknowledge Within</th>
              <th>Resolve Within</th>
              <th>Escalation Target</th>
              <th>Enabled</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {policies.map((policy) => (
              <tr key={policy.id}>
                <td><span className={`badge severity-${policy.severity.toLowerCase()}`}>{policy.severity}</span></td>
                <td>{policy.acknowledge_within_minutes} min</td>
                <td>{policy.resolve_within_minutes} min</td>
                <td className="wide-cell">{policy.escalation_target}</td>
                <td><span className={`badge ${policy.enabled ? "status-sent" : "status-failed"}`}>{policy.enabled ? "YES" : "NO"}</span></td>
                <td>
                  {canManagePolicies && (
                    <div className="table-actions">
                      <button className="secondary" onClick={() => editPolicy(policy)}>Edit</button>
                      <button className="danger" disabled={actionLoading === `delete-${policy.id}`} onClick={() => removePolicy(policy)}>Delete</button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
            {policies.length === 0 && <tr><td colSpan="6">No SLA policies found.</td></tr>}
          </tbody>
        </table>
      </section>

      <section className="actions-card">
        <div className="filters-row sla-filters">
          <label>
            Status
            <select name="status" value={filters.status} onChange={handleFilterChange}>
              <option value="">All</option>
              {statuses.map((status) => <option key={status} value={status}>{status}</option>)}
            </select>
          </label>
          <label>
            Severity
            <select name="severity" value={filters.severity} onChange={handleFilterChange}>
              <option value="">All</option>
              {severities.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
            </select>
          </label>
          <label>
            Incident ID
            <input name="incident_id" value={filters.incident_id} onChange={handleFilterChange} />
          </label>
        </div>
      </section>

      <section className="table-card">
        <h2>SLA Breaches</h2>
        <table>
          <thead>
            <tr>
              <th>Incident ID</th>
              <th>Severity</th>
              <th>SLA Status</th>
              <th>Acknowledge Due At</th>
              <th>Resolve Due At</th>
              <th>Acknowledged At</th>
              <th>Resolved At</th>
              <th>Escalated At</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {breaches.map((item) => (
              <tr key={item.id}>
                <td>{item.incident_id}</td>
                <td>{item.severity}</td>
                <td><span className={`badge sla-${item.status.toLowerCase()}`}>{item.status}</span></td>
                <td>{formatDate(item.acknowledge_due_at)}</td>
                <td>{formatDate(item.resolve_due_at)}</td>
                <td>{formatDate(item.acknowledged_at)}</td>
                <td>{formatDate(item.resolved_at)}</td>
                <td>{formatDate(item.escalated_at)}</td>
                <td>
                  <div className="table-actions">
                    <button onClick={() => viewEscalations(item.incident_id)}>View Escalations</button>
                    {canEscalate && <button className="secondary" onClick={() => viewEscalations(item.incident_id)}>Manual Escalate</button>}
                  </div>
                </td>
              </tr>
            ))}
            {breaches.length === 0 && <tr><td colSpan="9">No SLA breaches found.</td></tr>}
          </tbody>
        </table>
      </section>

      {selectedIncidentID && (
        <section className="card incident-panel">
          <div className="panel-header">
            <h2>Incident #{selectedIncidentID} Escalations</h2>
            <button className="secondary" onClick={() => setSelectedIncidentID("")}>Close</button>
          </div>

          {canEscalate && (
            <form className="form-grid manual-escalation-form" onSubmit={submitManualEscalation}>
              <label>
                Reason
                <input value={manualForm.reason} onChange={(event) => setManualForm((current) => ({ ...current, reason: event.target.value }))} />
              </label>
              <label>
                Target
                <input value={manualForm.target} onChange={(event) => setManualForm((current) => ({ ...current, target: event.target.value }))} required />
              </label>
              <label>
                Actor
                <input value={manualForm.actor} onChange={(event) => setManualForm((current) => ({ ...current, actor: event.target.value }))} />
              </label>
              <button type="submit" disabled={actionLoading === "manual-escalation"}>{actionLoading === "manual-escalation" ? "Escalating" : "Escalate"}</button>
            </form>
          )}

          <div className="timeline">
            {escalations.map((item) => (
              <div className="timeline-item" key={item.id}>
                <span className={`badge sla-${item.action.toLowerCase()}`}>{item.action}</span>
                <p>{item.reason || "No reason provided."}</p>
                <small>Target: {item.target || "-"} · Actor: {item.actor || "system"} · {formatDate(item.created_at)}</small>
              </div>
            ))}
            {escalations.length === 0 && <p className="empty-state">No escalation history found.</p>}
          </div>
        </section>
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
