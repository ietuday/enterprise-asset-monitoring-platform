import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import axios from "axios";
import App from "./App";

vi.mock("axios");

describe("App maintenance navigation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.pushState({}, "", "/");
    localStorage.setItem("token", "token");
    localStorage.setItem("user", JSON.stringify({ name: "Admin", role: "ADMIN" }));
    axios.create.mockReturnValue({
      get: vi.fn().mockResolvedValue({ data: [] }),
      post: vi.fn().mockResolvedValue({ data: {} }),
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the Maintenance nav item and page", async () => {
    render(<App />);

    const maintenanceButton = screen.getByRole("button", { name: /^Maintenance$/i });
    expect(maintenanceButton).toBeInTheDocument();
    fireEvent.click(maintenanceButton);

    await waitFor(() => expect(screen.getByRole("heading", { name: /^Maintenance$/i })).toBeInTheDocument());
  });

  it("renders maintenance insights from the dashboard API", async () => {
    const get = vi.fn((path) => {
      const responses = {
        "/api/reports/summary": {
          totalAssets: 1,
          totalAlerts: 0,
          openAlerts: 0,
          resolvedAlerts: 0,
          criticalAlerts: 0,
          highAlerts: 0,
        },
        "/api/assets": [],
        "/api/alerts": [],
        "/api/reports/asset-health": [],
        "/api/reports/maintenance-insights": [
          {
            asset_id: "1",
            asset_name: "Pump A",
            health_score: 45,
            risk_level: "high",
            open_tasks: 2,
            overdue_tasks: 1,
            last_maintenance_date: "2026-05-20",
            recommended_action: "Schedule preventive maintenance within 7 days",
          },
        ],
      };
      return Promise.resolve({ data: responses[path] });
    });
    axios.create.mockReturnValue({ get, post: vi.fn().mockResolvedValue({ data: {} }) });

    render(<App />);

    expect(await screen.findByRole("heading", { name: /Maintenance Insights/i })).toBeInTheDocument();
    expect(screen.getByText("Pump A")).toBeInTheDocument();
    expect(screen.getAllByText("high").find((element) => element.classList.contains("risk-high"))).toBeTruthy();
    expect(screen.getByText("Schedule preventive maintenance within 7 days")).toBeInTheDocument();
  });

  it("renders maintenance insights empty state", async () => {
    render(<App />);

    expect(await screen.findByText("No maintenance insights available")).toBeInTheDocument();
  });
});
