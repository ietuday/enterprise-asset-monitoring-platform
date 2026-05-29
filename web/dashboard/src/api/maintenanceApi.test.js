import axios from "axios";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  cancelMaintenanceTask,
  changeMaintenanceStatus,
  completeMaintenanceTask,
  createMaintenanceTask,
  listAssetHealth,
  listMaintenanceHistory,
  listMaintenanceTasks,
} from "./maintenanceApi";

vi.mock("axios");

describe("maintenanceApi", () => {
  beforeEach(() => {
    localStorage.setItem("token", "test-token");
    vi.clearAllMocks();
  });

  it("lists maintenance tasks with filters", async () => {
    axios.get.mockResolvedValue({ data: [{ id: 1 }] });

    await expect(listMaintenanceTasks({ status: "scheduled", priority: "" })).resolves.toEqual([{ id: 1 }]);

    expect(axios.get).toHaveBeenCalledWith("http://localhost:4000/api/maintenance/tasks", {
      headers: { Authorization: "Bearer test-token" },
      params: { status: "scheduled" },
    });
  });

  it("creates a maintenance task", async () => {
    const task = { title: "Inspect motor" };
    axios.post.mockResolvedValue({ data: { id: 2 } });

    await expect(createMaintenanceTask(task)).resolves.toEqual({ id: 2 });

    expect(axios.post).toHaveBeenCalledWith(
      "http://localhost:4000/api/maintenance/tasks",
      task,
      { headers: { Authorization: "Bearer test-token", "Content-Type": "application/json" } }
    );
  });

  it("changes, completes, and cancels task state", async () => {
    axios.patch.mockResolvedValue({ data: { status: "in_progress" } });
    axios.post.mockResolvedValueOnce({ data: { status: "completed" } }).mockResolvedValueOnce({ data: { status: "cancelled" } });

    await changeMaintenanceStatus(7, { status: "in_progress" });
    await completeMaintenanceTask(7, { comment: "done" });
    await cancelMaintenanceTask(7, { comment: "cancel" });

    expect(axios.patch).toHaveBeenCalledWith(
      "http://localhost:4000/api/maintenance/tasks/7/status",
      { status: "in_progress" },
      { headers: { Authorization: "Bearer test-token", "Content-Type": "application/json" } }
    );
    expect(axios.post).toHaveBeenCalledWith(
      "http://localhost:4000/api/maintenance/tasks/7/complete",
      { comment: "done" },
      { headers: { Authorization: "Bearer test-token", "Content-Type": "application/json" } }
    );
    expect(axios.post).toHaveBeenCalledWith(
      "http://localhost:4000/api/maintenance/tasks/7/cancel",
      { comment: "cancel" },
      { headers: { Authorization: "Bearer test-token", "Content-Type": "application/json" } }
    );
  });

  it("loads maintenance history and asset health", async () => {
    axios.get.mockResolvedValueOnce({ data: [{ action: "TASK_CREATED" }] }).mockResolvedValueOnce({ data: [{ asset_id: "motor-101" }] });

    await expect(listMaintenanceHistory(4)).resolves.toEqual([{ action: "TASK_CREATED" }]);
    await expect(listAssetHealth()).resolves.toEqual([{ asset_id: "motor-101" }]);

    expect(axios.get).toHaveBeenCalledWith("http://localhost:4000/api/maintenance/history/4", {
      headers: { Authorization: "Bearer test-token" },
    });
    expect(axios.get).toHaveBeenCalledWith("http://localhost:4000/api/reports/asset-health", {
      headers: { Authorization: "Bearer test-token" },
    });
  });
});
