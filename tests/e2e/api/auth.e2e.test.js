const { ApiClient } = require("./helpers/apiClient");

describe("auth API E2E", () => {
  it("logs in as admin and calls a protected endpoint", async () => {
    const api = new ApiClient();

    const login = await api.login();
    expect(login.token).toBeTruthy();
    expect(login.user.email).toBe("admin@example.com");

    const assets = await api.listAssets();
    expect(Array.isArray(assets)).toBe(true);
  });
});
