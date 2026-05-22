import { useEffect, useMemo, useState } from "react";
import axios from "axios";
import "./App.css";
import RulesPage from "./pages/RulesPage";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:4000";

function App() {
  const [token, setToken] = useState(localStorage.getItem("token") || "");
  const [user, setUser] = useState(() => {
    const storedUser = localStorage.getItem("user");
    return storedUser ? JSON.parse(storedUser) : null;
  });

  const [activePage, setActivePage] = useState("dashboard");

  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("admin123");

  const [summary, setSummary] = useState(null);
  const [assets, setAssets] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [error, setError] = useState("");

  const [telemetryForm, setTelemetryForm] = useState({
    assetId: "dynamic-ui-motor-101",
    temperature: 70,
    cpu: 95,
    memory: 50,
    status: "RUNNING",
  });

  const api = useMemo(() => {
    return axios.create({
      baseURL: API_BASE_URL,
      headers: token
        ? {
          Authorization: `Bearer ${token}`,
        }
        : {},
    });
  }, [token]);

  async function login(event) {
    event.preventDefault();
    setError("");

    try {
      const response = await axios.post(`${API_BASE_URL}/api/auth/login`, {
        email,
        password,
      });

      const receivedToken = response.data.token;
      const receivedUser = response.data.user;

      localStorage.setItem("token", receivedToken);
      localStorage.setItem("user", JSON.stringify(receivedUser));

      setToken(receivedToken);
      setUser(receivedUser);
      setActivePage("dashboard");
    } catch (err) {
      setError(err.response?.data?.error || "Login failed");
    }
  }

  function logout() {
    localStorage.removeItem("token");
    localStorage.removeItem("user");

    setToken("");
    setUser(null);
    setSummary(null);
    setAssets([]);
    setAlerts([]);
    setActivePage("dashboard");
  }

  async function loadDashboard() {
    try {
      setError("");

      const [summaryRes, assetsRes, alertsRes] = await Promise.all([
        api.get("/api/reports/summary"),
        api.get("/api/assets"),
        api.get("/api/alerts"),
      ]);

      setSummary(summaryRes.data);
      setAssets(assetsRes.data);
      setAlerts(alertsRes.data);
    } catch (err) {
      setError(err.response?.data?.error || "Failed to load dashboard");
    }
  }

  function handleTelemetryChange(event) {
    const { name, value } = event.target;

    setTelemetryForm((current) => ({
      ...current,
      [name]: value,
    }));
  }

  async function sendCustomTelemetry(event) {
    event.preventDefault();

    try {
      setError("");

      await api.post("/api/telemetry", {
        assetId: telemetryForm.assetId,
        temperature: Number(telemetryForm.temperature),
        cpu: Number(telemetryForm.cpu),
        memory: Number(telemetryForm.memory),
        status: telemetryForm.status,
      });

      await loadDashboard();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to send telemetry");
    }
  }

  async function sendAbnormalTelemetry() {
    try {
      setError("");

      await api.post("/api/telemetry", {
        assetId: "motor-101",
        temperature: 96,
        cpu: 65,
        memory: 50,
        status: "RUNNING",
      });

      await loadDashboard();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to send telemetry");
    }
  }

  async function sendNormalTelemetry() {
    try {
      setError("");

      await api.post("/api/telemetry", {
        assetId: "motor-101",
        temperature: 70,
        cpu: 60,
        memory: 50,
        status: "RUNNING",
      });

      await loadDashboard();
    } catch (err) {
      setError(err.response?.data?.error || "Failed to send telemetry");
    }
  }

  useEffect(() => {
    if (token && activePage === "dashboard") {
      loadDashboard();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, activePage]);

  if (!token) {
    return (
      <div className="page">
        <div className="login-card">
          <h1>Enterprise Asset Monitoring</h1>
          <p>Login to monitor assets, telemetry, alerts, reports, and dynamic rules.</p>

          <form onSubmit={login}>
            <label>Email</label>
            <input
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="admin@example.com"
            />

            <label>Password</label>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="admin123"
            />

            {error && <div className="error">{error}</div>}

            <button type="submit">Login</button>
          </form>
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <header className="header">
        <div>
          <h1>Enterprise Asset Monitoring</h1>
          <p>
            Secure microservices dashboard
            {user?.role ? ` · ${user.name} · ${user.role}` : ""}
          </p>
        </div>

        <div className="header-actions">
          <button
            className={activePage === "dashboard" ? "active" : "secondary"}
            onClick={() => setActivePage("dashboard")}
          >
            Dashboard
          </button>

          <button
            className={activePage === "rules" ? "active" : "secondary"}
            onClick={() => setActivePage("rules")}
          >
            Rules
          </button>

          {activePage === "dashboard" && (
            <button onClick={loadDashboard}>Refresh</button>
          )}

          <button className="secondary" onClick={logout}>
            Logout
          </button>
        </div>
      </header>

      {activePage === "rules" ? (
        <RulesPage />
      ) : (
        <DashboardPage
          summary={summary}
          assets={assets}
          alerts={alerts}
          error={error}
          telemetryForm={telemetryForm}
          handleTelemetryChange={handleTelemetryChange}
          sendCustomTelemetry={sendCustomTelemetry}
          sendAbnormalTelemetry={sendAbnormalTelemetry}
          sendNormalTelemetry={sendNormalTelemetry}
        />
      )}
    </div>
  );
}

function DashboardPage({
  summary,
  assets,
  alerts,
  error,
  telemetryForm,
  handleTelemetryChange,
  sendCustomTelemetry,
  sendAbnormalTelemetry,
  sendNormalTelemetry,
}) {
  return (
    <>
      {error && <div className="error">{error}</div>}

      {summary && (
        <section className="summary-grid">
          <SummaryCard title="Total Assets" value={summary.totalAssets} />
          <SummaryCard title="Total Alerts" value={summary.totalAlerts} />
          <SummaryCard title="Open Alerts" value={summary.openAlerts} />
          <SummaryCard title="Resolved Alerts" value={summary.resolvedAlerts} />
          <SummaryCard title="Critical Alerts" value={summary.criticalAlerts} />
          <SummaryCard title="High Alerts" value={summary.highAlerts} />
        </section>
      )}

      <section className="actions-card">
        <h2>Telemetry Simulator</h2>
        <p>
          Send custom telemetry to test static and dynamic monitoring rules.
        </p>

        <form className="telemetry-form" onSubmit={sendCustomTelemetry}>
          <label>
            Asset ID
            <input
              name="assetId"
              value={telemetryForm.assetId}
              onChange={handleTelemetryChange}
              required
            />
          </label>

          <label>
            Temperature
            <input
              name="temperature"
              type="number"
              value={telemetryForm.temperature}
              onChange={handleTelemetryChange}
              required
            />
          </label>

          <label>
            CPU
            <input
              name="cpu"
              type="number"
              value={telemetryForm.cpu}
              onChange={handleTelemetryChange}
              required
            />
          </label>

          <label>
            Memory
            <input
              name="memory"
              type="number"
              value={telemetryForm.memory}
              onChange={handleTelemetryChange}
              required
            />
          </label>

          <label>
            Status
            <select
              name="status"
              value={telemetryForm.status}
              onChange={handleTelemetryChange}
            >
              <option value="RUNNING">RUNNING</option>
              <option value="DOWN">DOWN</option>
              <option value="UNKNOWN">UNKNOWN</option>
            </select>
          </label>

          <button type="submit">Send Telemetry</button>
        </form>

        <div className="button-row quick-actions">
          <button onClick={sendAbnormalTelemetry}>
            Send Abnormal Temperature
          </button>
          <button className="secondary" onClick={sendNormalTelemetry}>
            Send Normal Temperature
          </button>
        </div>
      </section>
      <section className="content-grid">
        <div className="table-card">
          <h2>Assets</h2>

          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Type</th>
                <th>Location</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {assets.map((asset) => (
                <tr key={asset.id}>
                  <td>{asset.id}</td>
                  <td>{asset.name}</td>
                  <td>{asset.type}</td>
                  <td>{asset.location}</td>
                  <td>
                    <span className="badge">{asset.status}</span>
                  </td>
                </tr>
              ))}

              {assets.length === 0 && (
                <tr>
                  <td colSpan="5">No assets found.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="table-card">
          <h2>Alerts</h2>

          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Asset</th>
                <th>Name</th>
                <th>Severity</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {alerts.map((alert) => (
                <tr key={alert.id}>
                  <td>{alert.id}</td>
                  <td>{alert.assetId}</td>
                  <td>{alert.name}</td>
                  <td>
                    <span className={`badge severity-${alert.severity.toLowerCase()}`}>
                      {alert.severity}
                    </span>
                  </td>
                  <td>
                    <span className={`badge status-${alert.status.toLowerCase()}`}>
                      {alert.status}
                    </span>
                  </td>
                </tr>
              ))}

              {alerts.length === 0 && (
                <tr>
                  <td colSpan="5">No alerts found.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </>
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

export default App;