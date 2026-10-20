import "@testing-library/jest-dom";
import { beforeEach, vi } from "vitest";

// AuthProvider probes GET /api/me the moment it mounts, so any component
// rendered inside it would otherwise reach for the network. Default every
// test to "logged out"; a test that needs a signed-in user overrides fetch
// itself. Re-stubbed per test so one test's override can't leak.
beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => ({
      ok: false,
      status: 401,
      json: async () => null,
    })),
  );
});
