require("dotenv").config();

const { initDb } = require("./src/db/postgres");
const { auditLog } = require("./src/middleware/audit.middleware");

const express = require("express");
const cors = require("cors");
const helmet = require("helmet");
const morgan = require("morgan");
const jwt = require("jsonwebtoken");
const { createProxyMiddleware } = require("http-proxy-middleware");

// Initialize Express app
const app = express();

const PORT = process.env.PORT || 4000;

// Service URLs (can be set via environment variables or default to localhost)
const ASSET_SERVICE_URL =
  process.env.ASSET_SERVICE_URL || "http://localhost:5001";

const TELEMETRY_SERVICE_URL =
  process.env.TELEMETRY_SERVICE_URL || "http://localhost:5002";

const ALERT_SERVICE_URL =
  process.env.ALERT_SERVICE_URL || "http://localhost:5003";

const AUTH_SERVICE_URL =
  process.env.AUTH_SERVICE_URL || "http://localhost:4001";

const REPORT_SERVICE_URL =
  process.env.REPORT_SERVICE_URL || "http://localhost:8000";

const RULE_SERVICE_URL =
  process.env.RULE_SERVICE_URL || "http://localhost:5004";

const JWT_SECRET = process.env.JWT_SECRET || "supersecretkey";

// Middleware
app.use(helmet());
app.use(cors());
app.use(morgan("combined"));
app.use(auditLog);

app.get("/health", (req, res) => {
  res.status(200).json({
    service: "api-gateway",
    status: "healthy",
  });
});

/**
 * Authentication middleware
 * - Checks for Bearer token in Authorization header
 * - Verifies JWT and extracts user info
 * - Attaches user info to req.user and custom headers for downstream services
 */
function authenticate(req, res, next) {
  const authHeader = req.headers.authorization;

  if (!authHeader || !authHeader.startsWith("Bearer ")) {
    return res.status(401).json({
      error: "missing or invalid authorization header",
    });
  }

  const token = authHeader.split(" ")[1];

  try {
    const decoded = jwt.verify(token, JWT_SECRET);
    req.user = decoded;

    req.headers["x-user-id"] = decoded.id;
    req.headers["x-user-email"] = decoded.email;
    req.headers["x-user-role"] = decoded.role;

    next();
  } catch (err) {
    return res.status(401).json({
      error: "invalid or expired token",
    });
  }
}

/**
 * Authorization middleware factory
 * @param  {...string} allowedRoles - list of roles that are allowed to access the route
 * @returns middleware function that checks if the user's role is in the allowedRoles list
 */
function authorizeRoles(...allowedRoles) {
  return (req, res, next) => {
    const userRole = req.user?.role;

    if (!userRole) {
      return res.status(403).json({
        error: "user role missing",
      });
    }

    if (!allowedRoles.includes(userRole)) {
      return res.status(403).json({
        error: "access denied",
        requiredRoles: allowedRoles,
        currentRole: userRole,
      });
    }

    next();
  };
}

/**
 * Public Auth Routes
 *
 * Gateway:
 * POST /api/auth/register
 * POST /api/auth/login
 * GET  /api/auth/me
 *
 * Auth Service:
 * POST /auth/register
 * POST /auth/login
 * GET  /auth/me
 */
app.use(
  "/api/auth",
  createProxyMiddleware({
    target: AUTH_SERVICE_URL,
    changeOrigin: true,
    pathRewrite: (path) => {
      return path === "/" ? "/auth" : `/auth${path}`;
    },
  })
);

/**
 * Protected Asset Routes
 * - GET /api/assets -> ADMIN, OPERATOR, VIEWER
 * - POST/PUT/DELETE /api/assets -> ADMIN only
 */
app.use(
  "/api/assets",
  authenticate,
  (req, res, next) => {
    if (req.method === "GET") {
      return authorizeRoles("ADMIN", "OPERATOR", "VIEWER")(req, res, next);
    }

    return authorizeRoles("ADMIN")(req, res, next);
  },
  createProxyMiddleware({
    target: ASSET_SERVICE_URL,
    changeOrigin: true,
    pathRewrite: (path) => {
      return path === "/" ? "/assets" : `/assets${path}`;
    },
  })
);

/**
 * Protected Telemetry Routes
 * - GET /api/telemetry -> ADMIN, OPERATOR, VIEWER
 * - POST/PUT/DELETE /api/telemetry -> ADMIN only
 */
app.use(
  "/api/telemetry",
  authenticate,
  (req, res, next) => {
    if (req.method === "GET") {
      return authorizeRoles("ADMIN", "OPERATOR", "VIEWER")(req, res, next);
    }

    return authorizeRoles("ADMIN", "OPERATOR")(req, res, next);
  },
  createProxyMiddleware({
    target: TELEMETRY_SERVICE_URL,
    changeOrigin: true,
    pathRewrite: (path) => {
      return path === "/" ? "/telemetry" : `/telemetry${path}`;
    },
  })
);
/**
 * Protected Alert Routes
 * - GET /api/alerts -> ADMIN, OPERATOR, VIEWER
 * - POST/PUT/DELETE /api/alerts -> ADMIN, OPERATOR only
 */
app.use(
  "/api/alerts",
  authenticate,
  (req, res, next) => {
    if (req.method === "GET") {
      return authorizeRoles("ADMIN", "OPERATOR", "VIEWER")(req, res, next);
    }

    return authorizeRoles("ADMIN", "OPERATOR")(req, res, next);
  },
  createProxyMiddleware({
    target: ALERT_SERVICE_URL,
    changeOrigin: true,
    pathRewrite: (path) => {
      return path === "/" ? "/alerts" : `/alerts${path}`;
    },
  })
);

/**
 * Protected Report Routes
 * - GET /api/reports -> ADMIN, OPERATOR, VIEWER
 * - POST/PUT/DELETE /api/reports -> ADMIN only
 */
app.use(
  "/api/reports",
  authenticate,
  authorizeRoles("ADMIN", "OPERATOR", "VIEWER"),
  createProxyMiddleware({
    target: REPORT_SERVICE_URL,
    changeOrigin: true,
    pathRewrite: (path) => {
      return path === "/" ? "/reports" : `/reports${path}`;
    },
  })
);

/**
 * Protected Rule Routes 
 * - GET /api/rules -> ADMIN, OPERATOR, VIEWER
 * - POST/PUT/DELETE /api/rules -> ADMIN only
 */
app.use(
  "/api/rules",
  authenticate,
  (req, res, next) => {
    if (req.method === "GET") {
      return authorizeRoles("ADMIN", "OPERATOR", "VIEWER")(req, res, next);
    }

    return authorizeRoles("ADMIN")(req, res, next);
  },
  createProxyMiddleware({
    target: RULE_SERVICE_URL,
    changeOrigin: true,
    pathRewrite: (path) => {
      return path === "/" ? "/rules" : `/rules${path}`;
    },
  })
);

app.use((req, res) => {
  res.status(404).json({
    error: "route not found",
    path: req.originalUrl,
  });
});

async function startServer() {
  try {
    await initDb();

    app.listen(PORT, () => {
      console.log(`api-gateway running on port ${PORT}`);
      console.log(`auth-service: ${AUTH_SERVICE_URL}`);
      console.log(`asset-service: ${ASSET_SERVICE_URL}`);
      console.log(`telemetry-service: ${TELEMETRY_SERVICE_URL}`);
      console.log(`alert-service: ${ALERT_SERVICE_URL}`);
      console.log(`report-service: ${REPORT_SERVICE_URL}`);
      console.log(`rule-service: ${RULE_SERVICE_URL}`);
    });
  } catch (err) {
    console.error("failed to start api-gateway:", err);
    process.exit(1);
  }
}

startServer();