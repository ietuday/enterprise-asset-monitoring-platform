import { useEffect, useState } from "react";
import {
  createRule,
  deleteRule,
  getRuleHistory,
  getRules,
  updateRule,
} from "../api/rulesApi";

const emptyRule = {
  name: "",
  metric: "cpu",
  operator: ">",
  threshold: 90,
  value: "",
  severity: "HIGH",
  enabled: true,
};

export default function RulesPage() {
  const [rules, setRules] = useState([]);
  const [form, setForm] = useState(emptyRule);
  const [loading, setLoading] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [error, setError] = useState("");
  const [history, setHistory] = useState([]);
  const [showHistory, setShowHistory] = useState(false);

  const user = JSON.parse(localStorage.getItem("user") || "{}");
  const isAdmin = user?.role === "ADMIN";

  async function loadRules() {
    try {
      setLoading(true);
      setError("");

      const data = await getRules();
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
  }

  useEffect(() => {
    loadRules();
  }, []);

  function handleChange(event) {
    const { name, value, type, checked } = event.target;

    setForm((current) => {
      const updated = {
        ...current,
        [name]: type === "checkbox" ? checked : value,
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
      enabled: source.enabled,
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

  async function handleToggle(rule) {
    try {
      setError("");

      await updateRule(rule.id, {
        ...buildRulePayload(rule),
        enabled: !rule.enabled,
      });

      await refreshAfterChange();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to update rule");
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

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Dynamic Monitoring Rules</h1>
          <p>Manage DB-driven rules that generate Prometheus alert rules.</p>
        </div>

        <div className="button-row">
          <button onClick={loadRules}>Refresh</button>
          <button className="secondary" onClick={loadHistory}>
            History
          </button>
        </div>
      </div>

      {error && <div className="error">{error}</div>}

      {isAdmin && (
        <form className="card form-grid" onSubmit={handleCreate}>
          <h2>Create Rule</h2>

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

          <label className="checkbox-row">
            <input
              name="enabled"
              type="checkbox"
              checked={form.enabled}
              onChange={handleChange}
            />
            Enabled
          </label>

          <button type="submit">Create Rule</button>
        </form>
      )}

      {!isAdmin && (
        <div className="info">You have read-only access to monitoring rules.</div>
      )}

      <div className="card">
        <h2>Rules</h2>

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
                <th>Enabled</th>
                <th>Updated</th>
                {isAdmin && <th>Actions</th>}
              </tr>
            </thead>

            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td>{rule.id}</td>
                  <td>{rule.name}</td>
                  <td>{rule.metric}</td>
                  <td>{rule.operator}</td>
                  <td>{rule.metric === "status" ? "-" : rule.threshold}</td>
                  <td>{rule.metric === "status" ? rule.value || "-" : "-"}</td>
                  <td>{rule.severity}</td>
                  <td>{rule.enabled ? "Yes" : "No"}</td>
                  <td>{new Date(rule.updated_at).toLocaleString()}</td>

                  {isAdmin && (
                    <td>
                      <button onClick={() => handleToggle(rule)}>
                        {rule.enabled ? "Disable" : "Enable"}
                      </button>

                      <button
                        className="danger"
                        onClick={() => handleDelete(rule.id)}
                      >
                        Delete
                      </button>
                    </td>
                  )}
                </tr>
              ))}

              {rules.length === 0 && (
                <tr>
                  <td colSpan={isAdmin ? 10 : 9}>No rules found.</td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>

      {showHistory && (
        <div className="card">
          <div className="page-header">
            <div>
              <h2>Rule Audit History</h2>
              <p>Recent rule create, update, and delete activities.</p>
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
                      <span className={`badge action-${item.action.toLowerCase()}`}>
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