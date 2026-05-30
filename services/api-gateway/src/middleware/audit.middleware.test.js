const { resolveAction } = require("./audit.middleware");

describe("resolveAction maintenance audit mapping", () => {
  it("maps maintenance task actions", () => {
    expect(resolveAction("POST", "/api/maintenance/tasks")).toBe("MAINTENANCE_TASK_CREATED");
    expect(resolveAction("PUT", "/api/maintenance/tasks/1")).toBe("MAINTENANCE_TASK_UPDATED");
    expect(resolveAction("PATCH", "/api/maintenance/tasks/1/status")).toBe("MAINTENANCE_TASK_STATUS_CHANGED");
    expect(resolveAction("POST", "/api/maintenance/tasks/1/complete")).toBe("MAINTENANCE_TASK_COMPLETED");
    expect(resolveAction("POST", "/api/maintenance/tasks/1/cancel")).toBe("MAINTENANCE_TASK_CANCELLED");
    expect(resolveAction("GET", "/api/maintenance/tasks")).toBe("MAINTENANCE_TASK_VIEWED");
  });
});

describe("resolveAction maintenance insights audit mapping", () => {
  it("maps maintenance insights reads before generic report reads", () => {
    expect(resolveAction("GET", "/api/reports/maintenance-insights")).toBe(
      "MAINTENANCE_INSIGHTS_VIEWED"
    );
    expect(resolveAction("GET", "/api/reports/summary")).toBe("READ_REPORTS");
  });
});
