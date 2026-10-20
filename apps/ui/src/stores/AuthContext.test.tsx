import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, vi } from "vitest";
import { AuthProvider, useAuth } from "./AuthContext";
import { gameSocket } from "../lib/ws/client";

// The socket's lifecycle is a side effect of signing in, so it's spied on
// rather than driven for real here.
vi.mock("../lib/ws/client", () => ({
  gameSocket: {
    connect: vi.fn(),
    disconnect: vi.fn(),
    subscribe: vi.fn(() => () => {}),
    send: vi.fn(),
  },
}));

const user = { id: "user-1", email: "alice@example.com", username: "alice" };

function respond(body: unknown, ok = true, status = 200) {
  return { ok, status, json: async () => body };
}

/**
 * Stubs fetch with a sequence, repeating the last entry once exhausted.
 * The first entry answers the /api/me probe AuthProvider fires on mount.
 * Typed with its parameters so mock.calls stays inspectable.
 */
function stubFetchSequence(...responses: ReturnType<typeof respond>[]) {
  const queue = [...responses];
  let i = 0;
  // Typed via the generic rather than by declaring unused parameters, so
  // mock.calls stays inspectable without tripping no-unused-vars.
  const fetchMock = vi.fn<(url: string, init?: RequestInit) => Promise<ReturnType<typeof respond>>>(
    async () => queue[Math.min(i++, queue.length - 1)],
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function renderAuth() {
  return renderHook(() => useAuth(), { wrapper: AuthProvider });
}

/** Renders and waits for the initial probe to settle. */
async function renderSettled() {
  const view = renderAuth();
  await waitFor(() => expect(view.result.current.isLoading).toBe(false));
  return view;
}

beforeEach(() => {
  vi.mocked(gameSocket.connect).mockClear();
  vi.mocked(gameSocket.disconnect).mockClear();
});

describe("the initial /api/me probe", () => {
  it("asks the server who is signed in, with credentials", async () => {
    const fetchMock = stubFetchSequence(respond(null, false, 401));

    await renderSettled();

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain("/api/me");
    // The session is an httpOnly cookie, so this must be sent.
    expect(init?.credentials).toBe("include");
  });

  it("adopts the user the probe returns", async () => {
    stubFetchSequence(respond(user));

    const { result } = await renderSettled();

    expect(result.current.user?.username).toBe("alice");
  });

  it("stays anonymous when the probe says unauthorised", async () => {
    stubFetchSequence(respond(null, false, 401));

    const { result } = await renderSettled();

    expect(result.current.user).toBeNull();
  });

  // A network failure must settle to logged-out rather than hang forever.
  it("settles to anonymous when the probe rejects", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new Error("offline");
      }),
    );

    const { result } = await renderSettled();

    expect(result.current.user).toBeNull();
  });
});

describe("login", () => {
  it("posts the credentials then re-probes for the user", async () => {
    const fetchMock = stubFetchSequence(
      respond(null, false, 401), // probe
      respond(null), // POST /api/login
      respond(user), // re-probe
    );
    const { result } = await renderSettled();

    await act(async () => {
      await result.current.login({ email: user.email, password: "password123" });
    });

    expect(result.current.user?.username).toBe("alice");
    const [url, init] = fetchMock.mock.calls[1];
    expect(url).toContain("/api/login");
    expect(init?.method).toBe("POST");
    expect(init?.credentials).toBe("include");
  });

  it("throws on bad credentials and stays anonymous", async () => {
    stubFetchSequence(respond(null, false, 401), respond(null, false, 401));
    const { result } = await renderSettled();

    await expect(
      act(async () => {
        await result.current.login({ email: user.email, password: "wrong" });
      }),
    ).rejects.toThrow(/invalid email or password/i);

    expect(result.current.user).toBeNull();
  });
});

describe("logout", () => {
  it("posts to logout and clears the user", async () => {
    const fetchMock = stubFetchSequence(respond(user), respond(null));
    const { result } = await renderSettled();
    expect(result.current.user).not.toBeNull();

    await act(async () => {
      await result.current.logout();
    });

    expect(result.current.user).toBeNull();
    expect(fetchMock.mock.calls[1][0]).toContain("/api/logout");
  });
});

describe("the socket lifecycle", () => {
  it("connects once a user is signed in", async () => {
    stubFetchSequence(respond(user));

    await renderSettled();

    await waitFor(() => expect(gameSocket.connect).toHaveBeenCalled());
  });

  it("does not connect while nobody is signed in", async () => {
    stubFetchSequence(respond(null, false, 401));

    await renderSettled();

    expect(gameSocket.connect).not.toHaveBeenCalled();
    expect(gameSocket.disconnect).toHaveBeenCalled();
  });

  it("disconnects on logout", async () => {
    stubFetchSequence(respond(user), respond(null));
    const { result } = await renderSettled();
    await waitFor(() => expect(gameSocket.connect).toHaveBeenCalled());
    vi.mocked(gameSocket.disconnect).mockClear();

    await act(async () => {
      await result.current.logout();
    });

    expect(gameSocket.disconnect).toHaveBeenCalled();
  });
});

describe("useAuth outside a provider", () => {
  it("throws rather than handing back a silently empty context", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

    expect(() => renderHook(() => useAuth())).toThrow(
      /must be used within an AuthProvider/,
    );

    consoleError.mockRestore();
  });
});
