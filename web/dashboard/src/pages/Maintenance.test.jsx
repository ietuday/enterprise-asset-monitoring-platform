import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import MaintenancePage from "./Maintenance";
import {
  cancelMaintenanceTask,
  completeMaintenanceTask,
  createMaintenanceTask,
  listAssetHealth,
  listMaintenanceTasks,
} from "../api/maintenanceApi";

vi.mock("../api/maintenanceApi");

const task = {
  id: 1,
  asset_id: "motor-101",
  title: "Inspect motor",
  maintenance_type: "inspection",
  priority: "medium",
  status: "scheduled",
  due_date: new Date("2026-06-01T10:00:00Z").toISOString(),
  assigned_to: "operator@example.com",
};

const health = {
  asset_id: "motor-101",
  asset_name: "Motor 101",
  health_score: 85,
  health_status: "healthy",
  reasons: [],
};

describe("MaintenancePage", () => {
  beforeEach(() => {
    localStorage.setItem("user", JSON.stringify({ role: "ADMIN", email: "admin@example.com" }));
    listMaintenanceTasks.mockResolvedValue([task]);
    listAssetHealth.mockResolvedValue([health]);
    createMaintenanceTask.mockResolvedValue({ id: 2 });
    completeMaintenanceTask.mockResolvedValue({ ...task, status: "completed" });
    cancelMaintenanceTask.mockResolvedValue({ ...task, status: "cancelled" });
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("renders tasks, summary cards, and asset health", async () => {
    render(<MaintenancePage />);

    expect(await screen.findByRole("heading", { name: /^Maintenance$/ })).toBeInTheDocument();
    expect(screen.getByText("Scheduled")).toBeInTheDocument();
    expect(screen.getByText("In Progress")).toBeInTheDocument();
    expect(await screen.findByText("Inspect motor")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Asset Health/i })).toBeInTheDocument();
    expect(screen.getByText("Motor 101")).toBeInTheDocument();
    expect(screen.getByText("No active risk signals")).toBeInTheDocument();
  });

  it("submits a create task payload", async () => {
    render(<MaintenancePage />);
    await screen.findByText("Inspect motor");

    const form = screen.getAllByRole("heading", { name: /Create Maintenance Task/i })[0].closest("form");
    fireEvent.change(within(form).getByLabelText(/Asset ID/i), { target: { value: "pump-101" } });
    fireEvent.change(within(form).getByLabelText(/^Title$/i), { target: { value: "Inspect pump" } });
    fireEvent.change(within(form).getByLabelText(/^Type$/i), { target: { value: "inspection" } });
    fireEvent.click(within(form).getByRole("button", { name: /Create Task/i }));

    await waitFor(() => expect(createMaintenanceTask).toHaveBeenCalled());
    expect(createMaintenanceTask.mock.calls[0][0]).toMatchObject({
      asset_id: "pump-101",
      title: "Inspect pump",
      maintenance_type: "inspection",
      created_by: "admin@example.com",
    });
  });

  it("completes and cancels tasks", async () => {
    render(<MaintenancePage />);
    const row = (await screen.findAllByText("Inspect motor"))[0].closest("tr");

    fireEvent.click(within(row).getByRole("button", { name: /^Complete$/i }));
    await waitFor(() => expect(completeMaintenanceTask).toHaveBeenCalledWith(1, expect.objectContaining({ performed_by: "admin@example.com" })));

    fireEvent.click(within(row).getByRole("button", { name: /^Cancel$/i }));
    await waitFor(() => expect(cancelMaintenanceTask).toHaveBeenCalledWith(1, expect.objectContaining({ performed_by: "admin@example.com" })));
  });

  it("reloads when filters change", async () => {
    render(<MaintenancePage />);
    await screen.findAllByText("Inspect motor");

    const filters = document.querySelector(".maintenance-filters");
    fireEvent.change(within(filters).getByLabelText(/Status/i), { target: { value: "scheduled" } });

    await waitFor(() => expect(listMaintenanceTasks).toHaveBeenCalledWith(expect.objectContaining({ status: "scheduled" })));
  });

  it("shows API errors", async () => {
    listMaintenanceTasks.mockRejectedValueOnce({ response: { data: { error: "maintenance unavailable" } } });

    render(<MaintenancePage />);

    expect(await screen.findByText("maintenance unavailable")).toBeInTheDocument();
  });
});
