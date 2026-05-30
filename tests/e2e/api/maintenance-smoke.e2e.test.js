const { ApiClient } = require("./helpers/apiClient");
const { createUniqueAsset, uniqueSuffix } = require("./helpers/testData");

describe("maintenance API smoke", () => {
  it("creates, completes, and records history for a maintenance task", async () => {
    const api = new ApiClient();
    await api.login();

    const suffix = uniqueSuffix();
    const asset = createUniqueAsset(suffix);
    await api.createAsset(asset);

    const scheduledDate = new Date(Date.now() + 60 * 60 * 1000);
    const dueDate = new Date(Date.now() + 24 * 60 * 60 * 1000);
    const title = `E2E maintenance ${suffix}`;

    const created = await api.createMaintenanceTask({
      asset_id: asset.id,
      title,
      description: "E2E preventive maintenance smoke task",
      maintenance_type: "inspection",
      priority: "medium",
      scheduled_date: scheduledDate.toISOString(),
      due_date: dueDate.toISOString(),
      assigned_to: "operator@example.com",
      created_by: "admin@example.com",
    });

    expect(created.id).toBeTruthy();
    expect(created.status).toBe("scheduled");

    const tasks = await api.listMaintenanceTasks({ asset_id: asset.id });
    expect(tasks.some((task) => task.id === created.id && task.title === title)).toBe(true);

    const completed = await api.completeMaintenanceTask(created.id, {
      comment: "Maintenance completed by smoke test",
      performed_by: "operator@example.com",
    });
    expect(completed.status).toBe("completed");

    const history = await api.listMaintenanceHistory(created.id);
    const actions = history.map((item) => item.action);
    expect(actions).toContain("TASK_CREATED");
    expect(actions).toContain("TASK_COMPLETED");

    const healthRows = await api.listAssetHealth();
    expect(Array.isArray(healthRows)).toBe(true);

    const insights = await api.listMaintenanceInsights();
    expect(Array.isArray(insights)).toBe(true);
    if (insights.length > 0) {
      expect(insights[0]).toEqual(expect.objectContaining({
        asset_id: expect.any(String),
        asset_name: expect.any(String),
        health_score: expect.any(Number),
        risk_level: expect.any(String),
        open_tasks: expect.any(Number),
        overdue_tasks: expect.any(Number),
        recommended_action: expect.any(String),
        reason: expect.any(String),
      }));
    }
  });
});
