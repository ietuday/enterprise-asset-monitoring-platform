function uniqueSuffix() {
  return `${Date.now()}-${process.pid}-${Math.random().toString(36).slice(2, 10)}`;
}

function createUniqueAsset(suffix = uniqueSuffix()) {
  return {
    id: `e2e-motor-${suffix}`,
    name: `E2E Motor ${suffix}`,
    type: "MOTOR",
    location: "E2E Test Lab",
    status: "ACTIVE"
  };
}

function createCriticalTemperatureRule(suffix = uniqueSuffix()) {
  return {
    name: `E2E Critical Temperature ${suffix}`,
    metric: "temperature",
    operator: ">",
    threshold: 80,
    severity: "CRITICAL",
    enabled: true,
    status: "active"
  };
}

function createHighCpuRule(suffix = uniqueSuffix()) {
  return {
    name: `E2E High CPU ${suffix}`,
    metric: "cpu",
    operator: ">",
    threshold: 90,
    severity: "HIGH",
    enabled: true,
    status: "active"
  };
}

function createCriticalSLAPolicy() {
  return {
    severity: "CRITICAL",
    acknowledge_within_minutes: 1,
    resolve_within_minutes: 2,
    escalation_target: "manager@example.com",
    enabled: true
  };
}

function createWebhookChannel(suffix = uniqueSuffix(), port = process.env.E2E_WEBHOOK_PORT || 9100) {
  return {
    name: `E2E Webhook ${suffix}`,
    type: "WEBHOOK",
    target: `http://host.docker.internal:${port}/webhook`,
    enabled: true
  };
}

function createAbnormalTelemetry(assetId) {
  return {
    assetId,
    temperature: 95,
    cpu: 70,
    memory: 60,
    status: "RUNNING"
  };
}

function createNormalTelemetry(assetId) {
  return {
    assetId,
    temperature: 70,
    cpu: 50,
    memory: 45,
    status: "RUNNING"
  };
}

module.exports = {
  uniqueSuffix,
  createUniqueAsset,
  createCriticalTemperatureRule,
  createHighCpuRule,
  createCriticalSLAPolicy,
  createWebhookChannel,
  createAbnormalTelemetry,
  createNormalTelemetry
};
