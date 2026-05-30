const { pool } = require("../db/postgres");

function resolveAction(method, path) {
  if (path.startsWith("/api/assets")) {
    if (method === "POST") return "CREATE_ASSET";
    if (method === "PUT") return "UPDATE_ASSET";
    if (method === "DELETE") return "DELETE_ASSET";
    if (method === "GET") return "READ_ASSETS";
  }

  if (path.startsWith("/api/telemetry")) {
    if (method === "POST") return "SUBMIT_TELEMETRY";
    if (method === "GET") return "READ_TELEMETRY";
  }

  if (path.startsWith("/api/alerts")) {
    if (method === "POST") return "CREATE_ALERT";
    if (method === "PUT" && path.includes("acknowledge")) return "ACKNOWLEDGE_ALERT";
    if (method === "PUT" && path.includes("resolve")) return "RESOLVE_ALERT";
    if (method === "GET") return "READ_ALERTS";
  }

  if (path.startsWith("/api/reports/maintenance-insights")) {
    return "MAINTENANCE_INSIGHTS_VIEWED";
  }

  if (path.startsWith("/api/reports")) {
    return "READ_REPORTS";
  }

  if (path.startsWith("/api/maintenance")) {
    if (method === "POST" && path.includes("/complete")) return "MAINTENANCE_TASK_COMPLETED";
    if (method === "POST" && path.includes("/cancel")) return "MAINTENANCE_TASK_CANCELLED";
    if (method === "POST") return "MAINTENANCE_TASK_CREATED";
    if (method === "PUT") return "MAINTENANCE_TASK_UPDATED";
    if (method === "PATCH" && path.includes("/status")) return "MAINTENANCE_TASK_STATUS_CHANGED";
    if (method === "GET") return "MAINTENANCE_TASK_VIEWED";
  }

  if (path.startsWith("/api/auth/login")) {
    return "LOGIN";
  }

  if (path.startsWith("/api/auth/register")) {
    return "REGISTER_USER";
  }

  return "UNKNOWN";
}

function auditLog(req, res, next) {
  const startedAt = Date.now();

  res.on("finish", async () => {
    try {
      const action = resolveAction(req.method, req.originalUrl);

      await pool.query(
        `
        INSERT INTO audit_logs (
          user_id,
          user_email,
          user_role,
          method,
          path,
          status_code,
          action
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7);
        `,
        [
          req.user?.id || null,
          req.user?.email || null,
          req.user?.role || null,
          req.method,
          req.originalUrl,
          res.statusCode,
          action,
        ]
      );

      const duration = Date.now() - startedAt;
      console.log(
        `audit action=${action} method=${req.method} path=${req.originalUrl} status=${res.statusCode} duration=${duration}ms`
      );
    } catch (err) {
      console.error("failed to write audit log:", err.message);
    }
  });

  next();
}

module.exports = {
  auditLog,
  resolveAction,
};
