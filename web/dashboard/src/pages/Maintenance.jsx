import { useEffect, useMemo, useState } from "react";
import {
  cancelMaintenanceTask,
  changeMaintenanceStatus,
  completeMaintenanceTask,
  createMaintenanceTask,
  listAssetHealth,
  listMaintenanceTasks,
} from "../api/maintenanceApi";

const statuses = ["scheduled", "in_progress", "overdue", "completed", "cancelled"];
const priorities = ["low", "medium", "high", "critical"];

const emptyFilters = {
  status: "",
  priority: "",
  asset_id: "",
};

function defaultDate(offsetDays) {
  const value = new Date();
  value.setDate(value.getDate() + offsetDays);
  value.setMinutes(value.getMinutes() - value.getTimezoneOffset());
  return value.toISOString().slice(0, 16);
}

const emptyForm = {
  asset_id: "motor-101",
  title: "",
  description: "",
  maintenance_type: "inspection",
  priority: "medium",
  scheduled_date: defaultDate(0),
  due_date: defaultDate(7),
  assigned_to: "",
};

export default function MaintenancePage({ refreshSignal = 0 }) {
  const [tasks, setTasks] = useState([]);
  const [healthRows, setHealthRows] = useState([]);
  const [filters, setFilters] = useState(emptyFilters);
  const [form, setForm] = useState(emptyForm);
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState("");
  const [error, setError] = useState("");
  const [formError, setFormError] = useState("");
  const [notice, setNotice] = useState("");

  const user = JSON.parse(localStorage.getItem("user") || "{}");
  const canManage = user?.role === "ADMIN" || user?.role === "OPERATOR";
  const actor = user?.email || user?.name || "dashboard-user";

  const counts = useMemo(() => tasks.reduce(
    (current, task) => {
      current[task.status] = (current[task.status] || 0) + 1;
      return current;
    },
    { scheduled: 0, in_progress: 0, overdue: 0, completed: 0 }
  ), [tasks]);

  async function loadAll(nextFilters = filters) {
    try {
      setLoading(true);
      setError("");
      const [taskData, healthData] = await Promise.all([
        listMaintenanceTasks(nextFilters),
        listAssetHealth(),
      ]);
      setTasks(taskData);
      setHealthRows(healthData);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load maintenance data");
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

  function handleFormChange(event) {
    const { name, value } = event.target;
    setForm((current) => ({ ...current, [name]: value }));
    setFormError("");
  }

  function handleFilterChange(event) {
    const { name, value } = event.target;
    const nextFilters = { ...filters, [name]: value };
    setFilters(nextFilters);
    loadAll(nextFilters);
  }

  function validateForm() {
    if (!form.asset_id.trim()) return "Asset ID is required.";
    if (!form.title.trim()) return "Title is required.";
    if (!form.maintenance_type.trim()) return "Maintenance type is required.";
    if (!form.scheduled_date || !form.due_date) return "Scheduled and due dates are required.";
    if (new Date(form.due_date) < new Date(form.scheduled_date)) {
      return "Due date cannot be before scheduled date.";
    }
    return "";
  }

  async function submitTask(event) {
    event.preventDefault();
    const validation = validateForm();
    if (validation) {
      setFormError(validation);
      return;
    }

    try {
      setActionLoading("create");
      setNotice("");
      setError("");
      await createMaintenanceTask({
        ...form,
        asset_id: form.asset_id.trim(),
        title: form.title.trim(),
        maintenance_type: form.maintenance_type.trim(),
        scheduled_date: new Date(form.scheduled_date).toISOString(),
        due_date: new Date(form.due_date).toISOString(),
        created_by: actor,
      });
      setNotice("Maintenance task created.");
      setForm({ ...emptyForm, scheduled_date: defaultDate(0), due_date: defaultDate(7) });
      await loadAll();
    } catch (err) {
      setFormError(err.response?.data?.error || "Failed to create maintenance task");
    } finally {
      setActionLoading("");
    }
  }

  async function startTask(task) {
    await runTaskAction(`start-${task.id}`, "Maintenance task started.", () =>
      changeMaintenanceStatus(task.id, {
        status: "in_progress",
        comment: "Started from dashboard",
        performed_by: actor,
      })
    );
  }

  async function completeTask(task) {
    await runTaskAction(`complete-${task.id}`, "Maintenance task completed.", () =>
      completeMaintenanceTask(task.id, {
        comment: "Completed from dashboard",
        performed_by: actor,
      })
    );
  }

  async function cancelTask(task) {
    await runTaskAction(`cancel-${task.id}`, "Maintenance task cancelled.", () =>
      cancelMaintenanceTask(task.id, {
        comment: "Cancelled from dashboard",
        performed_by: actor,
      })
    );
  }

  async function runTaskAction(key, message, action) {
    try {
      setActionLoading(key);
      setNotice("");
      setError("");
      await action();
      setNotice(message);
      await loadAll();
    } catch (err) {
      setError(err.response?.data?.error || "Maintenance action failed");
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
          <h1>Maintenance</h1>
          <p>Schedule preventive work, track lifecycle state, and watch asset health.</p>
        </div>
        <button onClick={() => loadAll()} disabled={loading}>{loading ? "Refreshing" : "Refresh"}</button>
      </div>

      {error && <div className="error">{error}</div>}
      {notice && <div className="info">{notice}</div>}

      <section className="summary-grid maintenance-summary-grid">
        <SummaryCard title="Scheduled" value={counts.scheduled} />
        <SummaryCard title="In Progress" value={counts.in_progress} />
        <SummaryCard title="Overdue" value={counts.overdue} />
        <SummaryCard title="Completed" value={counts.completed} />
      </section>

      {canManage && (
        <section className="actions-card">
          <form className="form-grid maintenance-form" onSubmit={submitTask}>
            <h2>Create Maintenance Task</h2>
            <label>
              Asset ID
              <input name="asset_id" value={form.asset_id} onChange={handleFormChange} required />
            </label>
            <label>
              Title
              <input name="title" value={form.title} onChange={handleFormChange} placeholder="Quarterly pump inspection" required />
            </label>
            <label>
              Type
              <input name="maintenance_type" value={form.maintenance_type} onChange={handleFormChange} required />
            </label>
            <label>
              Priority
              <select name="priority" value={form.priority} onChange={handleFormChange}>
                {priorities.map((priority) => <option key={priority} value={priority}>{priority}</option>)}
              </select>
            </label>
            <label>
              Scheduled Date
              <input name="scheduled_date" type="datetime-local" value={form.scheduled_date} onChange={handleFormChange} required />
            </label>
            <label>
              Due Date
              <input name="due_date" type="datetime-local" value={form.due_date} onChange={handleFormChange} required />
            </label>
            <label>
              Assigned To
              <input name="assigned_to" value={form.assigned_to} onChange={handleFormChange} placeholder="operator@example.com" />
            </label>
            <label>
              Description
              <textarea name="description" value={form.description} onChange={handleFormChange} />
            </label>
            <div className="form-actions">
              <button type="submit" disabled={actionLoading === "create"}>
                {actionLoading === "create" ? "Creating" : "Create Task"}
              </button>
            </div>
            {formError && <div className="field-error">{formError}</div>}
          </form>
        </section>
      )}

      <section className="actions-card">
        <div className="filters-row maintenance-filters">
          <label>
            Status
            <select name="status" value={filters.status} onChange={handleFilterChange}>
              <option value="">All</option>
              {statuses.map((status) => <option key={status} value={status}>{status}</option>)}
            </select>
          </label>
          <label>
            Priority
            <select name="priority" value={filters.priority} onChange={handleFilterChange}>
              <option value="">All</option>
              {priorities.map((priority) => <option key={priority} value={priority}>{priority}</option>)}
            </select>
          </label>
          <label>
            Asset ID
            <input name="asset_id" value={filters.asset_id} onChange={handleFilterChange} placeholder="motor-101" />
          </label>
        </div>
      </section>

      <section className="table-card">
        <h2>Maintenance Tasks</h2>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Asset</th>
              <th>Title</th>
              <th>Type</th>
              <th>Priority</th>
              <th>Status</th>
              <th>Due Date</th>
              <th>Assigned To</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {tasks.map((task) => (
              <tr key={task.id}>
                <td>{task.id}</td>
                <td>{task.asset_id}</td>
                <td className="wide-cell">{task.title}</td>
                <td>{task.maintenance_type}</td>
                <td><span className={`badge priority-${task.priority}`}>{task.priority}</span></td>
                <td><span className={`badge maintenance-status-${task.status}`}>{task.status}</span></td>
                <td>{formatDate(task.due_date)}</td>
                <td>{task.assigned_to || "-"}</td>
                <td>
                  <div className="table-actions">
                    {canManage && task.status === "scheduled" && (
                      <button disabled={actionLoading === `start-${task.id}`} onClick={() => startTask(task)}>Start</button>
                    )}
                    {canManage && !["completed", "cancelled"].includes(task.status) && (
                      <button disabled={actionLoading === `complete-${task.id}`} onClick={() => completeTask(task)}>Complete</button>
                    )}
                    {canManage && !["completed", "cancelled"].includes(task.status) && (
                      <button className="danger" disabled={actionLoading === `cancel-${task.id}`} onClick={() => cancelTask(task)}>Cancel</button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
            {tasks.length === 0 && (
              <tr>
                <td colSpan="9">No maintenance tasks found.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>

      <section className="table-card">
        <h2>Asset Health</h2>
        <table>
          <thead>
            <tr>
              <th>Asset</th>
              <th>Health Score</th>
              <th>Health Status</th>
              <th>Reasons</th>
            </tr>
          </thead>
          <tbody>
            {healthRows.map((row) => (
              <tr key={row.asset_id}>
                <td>{row.asset_name || row.asset_id}</td>
                <td>{row.health_score}</td>
                <td><span className={`badge health-${row.health_status}`}>{row.health_status}</span></td>
                <td className="wide-cell">{row.reasons?.length ? row.reasons.join(", ") : "No active risk signals"}</td>
              </tr>
            ))}
            {healthRows.length === 0 && (
              <tr>
                <td colSpan="4">No asset health data found.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
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
