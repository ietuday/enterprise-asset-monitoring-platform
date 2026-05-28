describe.skip("slow SLA breach API E2E", () => {
  it("is reserved for nightly/manual SLA worker breach validation", () => {
    // Automatic breach validation waits for the SLA worker interval and real deadlines.
    // Keep it out of the default PR path; run manual smoke checks with a short
    // SLA_CHECK_INTERVAL_SECONDS value when needed.
  });
});
