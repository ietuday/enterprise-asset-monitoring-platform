async function waitFor({
  name,
  timeoutMs = 30000,
  intervalMs = 1000,
  action,
  predicate
}) {
  const startedAt = Date.now();
  let lastResult;
  let lastError;

  while (Date.now() - startedAt < timeoutMs) {
    try {
      lastResult = await action();
      if (predicate(lastResult)) {
        return lastResult;
      }
    } catch (err) {
      lastError = err;
    }

    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }

  const details = lastError
    ? lastError.message
    : JSON.stringify(lastResult, null, 2);

  throw new Error(`Timed out waiting for ${name}. Last result: ${details}`);
}

module.exports = {
  waitFor
};
