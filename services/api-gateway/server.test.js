const {
  authorizeMaintenanceRequest,
  authorizeReportRequest,
  reportRouteConfig,
} = require("./server");

describe("maintenance route RBAC", () => {
  it("allows viewer read access", () => {
    const { req, res, next } = requestParts("GET", "VIEWER");
    authorizeMaintenanceRequest(req, res, next);
    expect(next).toHaveBeenCalled();
    expect(res.status).not.toHaveBeenCalled();
  });

  it("blocks viewer write access", () => {
    const { req, res, next, body } = requestParts("POST", "VIEWER");
    authorizeMaintenanceRequest(req, res, next);
    expect(next).not.toHaveBeenCalled();
    expect(res.status).toHaveBeenCalledWith(403);
    expect(body.error).toBe("access denied");
  });

  it("allows operator and admin write requests", () => {
    for (const role of ["OPERATOR", "ADMIN"]) {
      const { req, res, next } = requestParts("POST", role);
      authorizeMaintenanceRequest(req, res, next);
      expect(next).toHaveBeenCalled();
      expect(res.status).not.toHaveBeenCalled();
    }
  });
});

describe("report route RBAC and maintenance insights routing", () => {
  it("allows viewer read access to reports", () => {
    const { req, res, next } = requestParts("GET", "VIEWER");
    authorizeReportRequest(req, res, next);
    expect(next).toHaveBeenCalled();
    expect(res.status).not.toHaveBeenCalled();
  });

  it("blocks viewer write access to reports", () => {
    const { req, res, next, body } = requestParts("POST", "VIEWER");
    authorizeReportRequest(req, res, next);
    expect(next).not.toHaveBeenCalled();
    expect(res.status).toHaveBeenCalledWith(403);
    expect(body.error).toBe("access denied");
  });

  it("registers maintenance insights under the report-service proxy", () => {
    expect(reportRouteConfig.prefix).toBe("/api/reports");
    expect(reportRouteConfig.downstreamPrefix).toBe("/reports");
    expect(reportRouteConfig.maintenanceInsightsPath).toBe("/maintenance-insights");
    expect(reportRouteConfig.target).toBeTruthy();
  });
});

function requestParts(method, role) {
  const body = {};
  const res = {
    status: vi.fn(() => res),
    json: vi.fn((payload) => Object.assign(body, payload)),
  };

  return {
    req: { method, user: { role } },
    res,
    next: vi.fn(),
    body,
  };
}
