import { useEffect, useMemo, useState } from "react";
import {
  activateRule,
  archiveRule,
  createRule,
  deleteRule,
  disableRule,
  getRuleHistory,
  getRules,
  getRuleVersions,
  rollbackRule,
  updateRule,
} from "../api/rulesApi";

const emptyRule = {
  name: "",
  metric: "cpu",
  operator: ">",
  threshold: 90,
  value: "",
  severity: "HIGH",
  status: "draft",
};

const statusOptions = ["all", "draft", "active", "disabled", "archived"];

export default function RulesPage({ refreshSignal = 0 }) {
  const [rules, setRules] = useState([]);
  const [form, setForm] = useState(emptyRule);
  const [editingRule, setEditingRule] = useState(null);
  const [statusFilter, setStatusFilter] = useState("all");
  const [loading, setLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [actionLoadingId, setActionLoadingId] = useState("");
  const [error, setError] = useState("");
  const [history, setHistory] = useState([]);
  const [showHistory, setShowHistory] = useState(false);
  const [versions, setVersions] = useState([]);
  const [selectedVersionRule, setSelectedVersionRule] = useState(null);

  const user = JSON.parse(localStorage.getItem("user") || "{}");
  const isAdmin = user?.role === "ADMIN";

  const lifecycleCounts = useMemo(() => {
    return rules.reduce(
      (counts, rule) => {
        const status = rule.status || "draft";
        counts[status] = (counts[status] || 0) + 1;
        counts.all += 1;
        return counts;
      },
      {
        all: 0,
        draft: 0,
        active: 0,
        disabled: 0,
        archived: 0,
      }
    );
  }, [rules]);

  async function loadRules(nextStatus = statusFilter) {
    try {
      setLoading(true);
      setError("");

      const data = await getRules(nextStatus === "all" ? "" : nextStatus);
      setRules(data);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load rules");
    } finally {
      setLoading(false);
    }
  }

  async function loadHistory() {
    try {
      setHistoryLoading(true);
      setError("");

      const data = await getRuleHistory();
      setHistory(data);
      setShowHistory(true);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load rule history");
    } finally {
      setHistoryLoading(false);
    }
  }

  async function refreshAfterChange() {
    await loadRules();

    if (showHistory) {
      await loadHistory();
    }

    if (selectedVersionRule) {
      await loadVersions(selectedVersionRule);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadRules();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (refreshSignal > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      loadRules();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshSignal]);

  function handleStatusFilterChange(event) {
    const nextStatus = event.target.value;
    setStatusFilter(nextStatus);
    loadRules(nextStatus);
  }

  function handleChange(event) {
    const { name, value } = event.target;

    setForm((current) => {
      const updated = {
        ...current,
        [name]: value,
      };

      if (name === "metric") {
        if (value === "status") {
          updated.operator = "==";
          updated.threshold = 0;
          updated.value = "DOWN";
          updated.severity = "CRITICAL";
        } else {
          updated.operator = ">";
          updated.threshold = current.threshold || 90;
          updated.value = "";
        }
      }

      return updated;
    });
  }

  function buildRulePayload(source) {
    return {
      name: source.name,
      metric: source.metric,
      operator: source.operator,
      threshold: source.metric === "status" ? 0 : Number(source.threshold),
      value: source.metric === "status" ? source.value : "",
      severity: source.severity,
      status: source.status || "draft",
    };
  }

  async function handleCreate(event) {
    event.preventDefault();

    try {
      setError("");

      await createRule(buildRulePayload(form));

      setForm(emptyRule);
      await refreshAfterChange();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to create rule");
    }
  }

  function startEdit(rule) {
    setEditingRule(rule);
    setForm({
      name: rule.name,
      metric: rule.metric,
      operator: rule.operator,
      threshold: rule.threshold,
      value: rule.value || "",
      severity: rule.severity,
      status: rule.status || "draft",
    });

    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function cancelEdit() {
    setEditingRule(null);
    setForm(emptyRule);
  }

  async function handleSave(event) {
    event.preventDefault();

    if (!editingRule) {
      await handleCreate(event);
      return;
    }

    try {
      setError("");

      await updateRule(editingRule.id, buildRulePayload(form));

      setEditingRule(null);
      setForm(emptyRule);
      await refreshAfterChange();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to update rule");
    }
  }

  async function handleLifecycleAction(rule, action) {
    const actionLabels = {
      activate: "activate",
      disable: "disable",
      archive: "archive",
    };

    const confirmed = window.confirm(
      `Are you sure you want to ${actionLabels[action]} rule "${rule.name}"?`
    );

    if (!confirmed) return;

    try {
      setError("");
      setActionLoadingId(`${action}-${rule.id}`);

      if (action === "activate") {
        await activateRule(rule.id);
      }

      if (action === "disable") {
        await disableRule(rule.id);
      }

      if (action === "archive") {
        await archiveRule(rule.id);
      }

      await refreshAfterChange();
    } catch (err) {
      setError(err.response?.data?.error || `Failed to ${action} rule`);
    } finally {
      setActionLoadingId("");
    }
  }

  async function handleDelete(id) {
    const confirmed = window.confirm("Delete this rule?");
    if (!confirmed) return;

    try {
      setError("");

      await deleteRule(id);
      await refreshAfterChange();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to delete rule");
    }
  }

  async function loadVersions(rule) {
    try {
      setVersionsLoading(true);
      setError("");

      const data = await getRuleVersions(rule.id);
      setVersions(data);
      setSelectedVersionRule(rule);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load rule versions");
    } finally {
      setVersionsLoading(false);
    }
  }

  async function handleRollback(version) {
    if (!selectedVersionRule) return;

    const confirmed = window.confirm(
      `Rollback rule "${selectedVersionRule.name}" to version ${version.version}?`
    );

    if (!confirmed) return;

    try {
      setError("");
      setActionLoadingId(`rollback-${version.version}`);

      await rollbackRule(selectedVersionRule.id, version.version);

      await refreshAfterChange();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to rollback rule");
    } finally {
      setActionLoadingId("");
    }
  }

  function statusBadgeClass(status) {
    return `badge rule-status-${status || "draft"}`;
  }

  function actionBadgeClass(action) {
    return `badge action-${String(action || "").toLowerCase()}`;
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Dynamic Monitoring Rules</h1>
          <p>Manage lifecycle, versions, rollback, and Prometheus rule generation.</p>
        </div>

        <div className="button-row">
          <button onClick={() => loadRules()}>Refresh</button>
          <button className="secondary" onClick={loadHistory}>
            History
          </button>
        </div>
      </div>

      {error && <div className="error">{error}</div>}

      <div className="summary-grid rules-summary-grid">
        {statusOptions.map((status) => (
          <div className="summary-card compact-summary" key={status}>
            <p>{status === "all" ? "Total Rules" : status}</p>
            <h2>{lifecycleCounts[status] || 0}</h2>
          </div>
        ))}
      </div>

      {isAdmin && (
        <form className="card form-grid" onSubmit={handleSave}>
          <h2>{editingRule ? `Edit Rule #${editingRule.id}` : "Create Rule"}</h2>

          <label>
            Name
            <input
              name="name"
              value={form.name}
              onChange={handleChange}
              placeholder={
                form.metric === "status"
                  ? "Dynamic Device Down"
                  : "Dynamic High CPU"
              }
              required
            />
          </label>

          <label>
            Metric
            <select name="metric" value={form.metric} onChange={handleChange}>
              <option value="temperature">temperature</option>
              <option value="cpu">cpu</option>
              <option value="memory">memory</option>
              <option value="status">status</option>
            </select>
          </label>

          <label>
            Operator
            <select name="operator" value={form.operator} onChange={handleChange}>
              {form.metric === "status" ? (
                <>
                  <option value="==">{"=="}</option>
                  <option value="!=">{"!="}</option>
                </>
              ) : (
                <>
                  <option value=">">{">"}</option>
                  <option value=">=">{">="}</option>
                  <option value="<">{"<"}</option>
                  <option value="<=">{"<="}</option>
                  <option value="==">{"=="}</option>
                  <option value="!=">{"!="}</option>
                </>
              )}
            </select>
          </label>

          {form.metric === "status" ? (
            <label>
              Value
              <select
                name="value"
                value={form.value}
                onChange={handleChange}
                required
              >
                <option value="DOWN">DOWN</option>
                <option value="RUNNING">RUNNING</option>
                <option value="UNKNOWN">UNKNOWN</option>
              </select>
            </label>
          ) : (
            <label>
              Threshold
              <input
                name="threshold"
                type="number"
                value={form.threshold}
                onChange={handleChange}
                required
              />
            </label>
          )}

          <label>
            Severity
            <select name="severity" value={form.severity} onChange={handleChange}>
              <option value="CRITICAL">CRITICAL</option>
              <option value="HIGH">HIGH</option>
              <option value="MEDIUM">MEDIUM</option>
              <option value="LOW">LOW</option>
            </select>
          </label>

          <label>
            Status
            <select name="status" value={form.status} onChange={handleChange}>
              <option value="draft">draft</option>
              <option value="active">active</option>
              <option value="disabled">disabled</option>
            </select>
          </label>

          <div className="button-row form-actions">
            <button type="submit">
              {editingRule ? "Save Changes" : "Create Rule"}
            </button>

            {editingRule && (
              <button type="button" className="secondary" onClick={cancelEdit}>
                Cancel
              </button>
            )}
          </div>
        </form>
      )}

      {!isAdmin && (
        <div className="info">You have read-only access to monitoring rules.</div>
      )}

      <div className="card">
        <div className="page-header">
          <div>
            <h2>Rules</h2>
            <p>Only active rules are included in Prometheus dynamic rule generation.</p>
          </div>

          <label className="inline-filter">
            Filter by status
            <select value={statusFilter} onChange={handleStatusFilterChange}>
              {statusOptions.map((status) => (
                <option key={status} value={status}>
                  {status}
                </option>
              ))}
            </select>
          </label>
        </div>

        {loading ? (
          <p>Loading rules...</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Metric</th>
                <th>Operator</th>
                <th>Threshold</th>
                <th>Value</th>
                <th>Severity</th>
                <th>Status</th>
                <th>Enabled</th>
                <th>Updated</th>
                <th>Versions</th>
                {isAdmin && <th>Actions</th>}
              </tr>
            </thead>

            <tbody>
              {rules.map((rule) => {
                const archived = rule.status === "archived";
                const active = rule.status === "active";
                const disabled = rule.status === "disabled";
                const draft = !rule.status || rule.status === "draft";

                return (
                  <tr key={rule.id}>
                    <td>{rule.id}</td>
                    <td>{rule.name}</td>
                    <td>{rule.metric}</td>
                    <td>{rule.operator}</td>
                    <td>{rule.metric === "status" ? "-" : rule.threshold}</td>
                    <td>{rule.metric === "status" ? rule.value || "-" : "-"}</td>
                    <td>{rule.severity}</td>
                    <td>
                      <span className={statusBadgeClass(rule.status)}>
                        {rule.status || "draft"}
                      </span>
                    </td>
                    <td>{rule.enabled ? "Yes" : "No"}</td>
                    <td>{new Date(rule.updated_at).toLocaleString()}</td>
                    <td>
                      <button className="secondary" onClick={() => loadVersions(rule)}>
                        Versions
                      </button>
                    </td>

                    {isAdmin && (
                      <td>
                        <div className="table-actions">
                          {!active && !archived && (
                            <button
                              disabled={actionLoadingId === `activate-${rule.id}`}
                              onClick={() => handleLifecycleAction(rule, "activate")}
                            >
                              Activate
                            </button>
                          )}

                          {active && (
                            <button
                              className="secondary"
                              disabled={actionLoadingId === `disable-${rule.id}`}
                              onClick={() => handleLifecycleAction(rule, "disable")}
                            >
                              Disable
                            </button>
                          )}

                          {(draft || disabled) && !archived && (
                            <button
                              className="secondary"
                              disabled={actionLoadingId === `archive-${rule.id}`}
                              onClick={() => handleLifecycleAction(rule, "archive")}
                            >
                              Archive
                            </button>
                          )}

                          {!archived && (
                            <button className="secondary" onClick={() => startEdit(rule)}>
                              Edit
                            </button>
                          )}

                          <button
                            className="danger"
                            onClick={() => handleDelete(rule.id)}
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    )}
                  </tr>
                );
              })}

              {rules.length === 0 && (
                <tr>
                  <td colSpan={isAdmin ? 12 : 11}>No rules found.</td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {selectedVersionRule && (
        <div className="card">
          <div className="page-header">
            <div>
              <h2>Rule Versions</h2>
              <p>
                Version history for rule #{selectedVersionRule.id} ·{" "}
                {selectedVersionRule.name}
              </p>
            </div>

            <div className="button-row">
              <button
                className="secondary"
                onClick={() => loadVersions(selectedVersionRule)}
              >
                Refresh Versions
              </button>

              <button
                className="secondary"
                onClick={() => {
                  setSelectedVersionRule(null);
                  setVersions([]);
                }}
              >
                Hide
              </button>
            </div>
          </div>

          {versionsLoading ? (
            <p>Loading versions...</p>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Version</th>
                  <th>Name</th>
                  <th>Metric</th>
                  <th>Operator</th>
                  <th>Threshold</th>
                  <th>Value</th>
                  <th>Severity</th>
                  <th>Status</th>
                  <th>Created By</th>
                  <th>Created At</th>
                  {isAdmin && <th>Actions</th>}
                </tr>
              </thead>

              <tbody>
                {versions.map((version) => (
                  <tr key={version.id}>
                    <td>{version.version}</td>
                    <td>{version.name}</td>
                    <td>{version.metric}</td>
                    <td>{version.operator}</td>
                    <td>{version.metric === "status" ? "-" : version.threshold}</td>
                    <td>{version.metric === "status" ? version.value || "-" : "-"}</td>
                    <td>{version.severity}</td>
                    <td>
                      <span className={statusBadgeClass(version.status)}>
                        {version.status || "draft"}
                      </span>
                    </td>
                    <td>{version.created_by || "system"}</td>
                    <td>{new Date(version.created_at).toLocaleString()}</td>
                    {isAdmin && (
                      <td>
                        <button
                          disabled={
                            selectedVersionRule.status === "archived" ||
                            actionLoadingId === `rollback-${version.version}`
                          }
                          onClick={() => handleRollback(version)}
                        >
                          Rollback
                        </button>
                      </td>
                    )}
                  </tr>
                ))}

                {versions.length === 0 && (
                  <tr>
                    <td colSpan={isAdmin ? 11 : 10}>
                      No versions found. Versions are created when a rule is updated.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          )}
        </div>
      )}

      {showHistory && (
        <div className="card">
          <div className="page-header">
            <div>
              <h2>Rule Audit History</h2>
              <p>Recent rule lifecycle, versioning, rollback, and delete activities.</p>
            </div>

            <div className="button-row">
              <button className="secondary" onClick={loadHistory}>
                Refresh History
              </button>

              <button className="secondary" onClick={() => setShowHistory(false)}>
                Hide
              </button>
            </div>
          </div>

          {historyLoading ? (
            <p>Loading history...</p>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Rule ID</th>
                  <th>Action</th>
                  <th>Rule Name</th>
                  <th>Changed By</th>
                  <th>Created At</th>
                </tr>
              </thead>

              <tbody>
                {history.map((item) => (
                  <tr key={item.id}>
                    <td>{item.id}</td>
                    <td>{item.rule_id || "-"}</td>
                    <td>
                      <span className={actionBadgeClass(item.action)}>
                        {item.action}
                      </span>
                    </td>
                    <td>{item.rule_name}</td>
                    <td>{item.changed_by || "system"}</td>
                    <td>{new Date(item.created_at).toLocaleString()}</td>
                  </tr>
                ))}

                {history.length === 0 && (
                  <tr>
                    <td colSpan="6">No rule history found.</td>
                  </tr>
                )}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}
