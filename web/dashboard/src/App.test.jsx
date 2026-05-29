import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import axios from "axios";
import App from "./App";

vi.mock("axios");

describe("App maintenance navigation", () => {
  beforeEach(() => {
    localStorage.setItem("token", "token");
    localStorage.setItem("user", JSON.stringify({ name: "Admin", role: "ADMIN" }));
    axios.create.mockReturnValue({
      get: vi.fn().mockResolvedValue({ data: [] }),
      post: vi.fn().mockResolvedValue({ data: {} }),
    });
  });

  it("renders the Maintenance nav item and page", async () => {
    render(<App />);

    const maintenanceButton = screen.getByRole("button", { name: /^Maintenance$/i });
    expect(maintenanceButton).toBeInTheDocument();
    fireEvent.click(maintenanceButton);

    await waitFor(() => expect(screen.getByRole("heading", { name: /^Maintenance$/i })).toBeInTheDocument());
  });
});
