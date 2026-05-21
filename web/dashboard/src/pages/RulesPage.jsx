import { useEffect, useState } from "react";
import { createRule, deleteRule, getRules, updateRule } from "../api/rulesApi";

const emptyRule = {
  name: "",
  metric: "cpu",
  operator: ">",
  threshold: 90,
  severity: "HIGH",
  enabled: true,
};

export default function RulesPage() {
  const [rules, setRules] = useState([]);
  const [form, setForm] = useState(emptyRule);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

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

  useEffect(() => {
    loadRules();
  }, []);

  function handleChange(event) {
    const { name, value, type, checked } = event.target;

    setForm((current) => ({
      ...current,
      [name]: type === "checkbox" ? checked : value,
    }));
  }

  async function handleCreate(event) {
    event.preventDefault();

    try {
      setError("");

      await createRule({
        ...form,
        threshold: Number(form.threshold),
      });

      setForm(emptyRule);
      await loadRules();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to create rule");
    }
  }

  async function handleToggle(rule) {
    try {
      setError("");

      await updateRule(rule.id, {
        name: rule.name,
        metric: rule.metric,
        operator: rule.operator,
        threshold: Number(rule.threshold),
        severity: rule.severity,
        enabled: !rule.enabled,
      });

      await loadRules();
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
      await loadRules();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to delete rule");
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>Dynamic Monitoring Rules</h1>
          <p>
            Manage DB-driven rules that generate Prometheus alert rules.
          </p>
        </div>
        <button onClick={loadRules}>Refresh</button>
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
              placeholder="Dynamic High CPU"
              required
            />
          </label>

          <label>
            Metric
            <select name="metric" value={form.metric} onChange={handleChange}>
              <option value="temperature">temperature</option>
              <option value="cpu">cpu</option>
              <option value="memory">memory</option>
            </select>
          </label>

          <label>
            Operator
            <select name="operator" value={form.operator} onChange={handleChange}>
              <option value=">">{">"}</option>
              <option value=">=">{">="}</option>
              <option value="<">{"<"}</option>
              <option value="<=">{"<="}</option>
              <option value="==">{"=="}</option>
              <option value="!=">{"!="}</option>
            </select>
          </label>

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
        <div className="info">
          You have read-only access to monitoring rules.
        </div>
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
                  <td>{rule.threshold}</td>
                  <td>{rule.severity}</td>
                  <td>{rule.enabled ? "Yes" : "No"}</td>
                  <td>{new Date(rule.updated_at).toLocaleString()}</td>
                  {isAdmin && (
                    <td>
                      <button onClick={() => handleToggle(rule)}>
                        {rule.enabled ? "Disable" : "Enable"}
                      </button>
                      <button className="danger" onClick={() => handleDelete(rule.id)}>
                        Delete
                      </button>
                    </td>
                  )}
                </tr>
              ))}

              {rules.length === 0 && (
                <tr>
                  <td colSpan={isAdmin ? 9 : 8}>No rules found.</td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}