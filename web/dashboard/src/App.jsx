import { useEffect, useState } from "react";
import axios from "axios";
import "./App.css";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:4000";

function App() {
  const [token, setToken] = useState(localStorage.getItem("token") || "");
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("admin123");

  const [summary, setSummary] = useState(null);
  const [assets, setAssets] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [error, setError] = useState("");

  const api = axios.create({
    baseURL: API_BASE_URL,
    headers: token
      ? {
          Authorization: `Bearer ${token}`,
        }
      : {},
  });

  async function login(event) {
    event.preventDefault();
    setError("");

    try {
      const response = await axios.post(`${API_BASE_URL}/api/auth/login`, {
        email,
        password,
      });

      const receivedToken = response.data.token;
      localStorage.setItem("token", receivedToken);
      setToken(receivedToken);
    } catch (err) {
      setError(err.response?.data?.error || "Login failed");
    }
  }

  function logout() {
    localStorage.removeItem("token");
    setToken("");
    setSummary(null);
    setAssets([]);
    setAlerts([]);
  }

  async function loadDashboard() {
    try {
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

  async function sendAbnormalTelemetry() {
    try {
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
    if (token) {
      loadDashboard();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  if (!token) {
    return (
      <div className="page">
        <div className="login-card">
          <h1>Enterprise Asset Monitoring</h1>
          <p>Login to monitor assets, telemetry, alerts, and reports.</p>

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
          <p>Secure microservices dashboard</p>
        </div>

        <div className="header-actions">
          <button onClick={loadDashboard}>Refresh</button>
          <button className="secondary" onClick={logout}>
            Logout
          </button>
        </div>
      </header>

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
        <p>Use these buttons to test alert creation and auto-resolution for motor-101.</p>

        <div className="button-row">
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
            </tbody>
          </table>
        </div>
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

export default App;