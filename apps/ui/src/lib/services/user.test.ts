import { beforeEach, vi } from "vitest";
import { checkEmailAvailable, checkUsernameAvailable, registerUser } from "./user";

let fetchMock: ReturnType<typeof vi.fn>;

function respond(body: unknown, ok = true, status = 200) {
  return { ok, status, json: async () => body, text: async () => (body == null ? "" : JSON.stringify(body)) };
}

/** http.Error's answer: text/plain with a trailing newline. */
function respondText(text: string, status: number) {
  return { ok: false, status, text: async () => text };
}

beforeEach(() => {
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

const registration = {
  username: "alice",
  email: "alice@example.com",
  password: "password123",
  rating: 800,
};

describe("registerUser", () => {
  it("posts the registration as JSON", async () => {
    fetchMock.mockResolvedValue(respond({ id: "user-1", username: "alice", rating: 800 }, true, 201));

    const result = await registerUser(registration);

    expect(result).toEqual({ id: "user-1", username: "alice", rating: 800 });
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/register");
    expect(options.method).toBe("POST");
    expect(options.headers).toMatchObject({ "Content-Type": "application/json" });
    expect(JSON.parse(options.body as string)).toEqual(registration);
  });

  // http.Error sends text/plain, so the message is the body itself.
  it("throws the server's plain-text message when the request fails", async () => {
    fetchMock.mockResolvedValue(respondText("username already taken\n", 409));

    await expect(registerUser(registration)).rejects.toThrow(/^username already taken$/);
  });

  it("falls back to a generic message when the body is empty", async () => {
    fetchMock.mockResolvedValue(respondText("", 500));

    await expect(registerUser(registration)).rejects.toThrow("Registration failed");
  });
});

// These two read a 404 as "that name is free". They previously requested
// /api/users/... while the router registers /api/user/... (singular), so
// every request 404'd and every name looked available. The URL is asserted
// here to keep that from regressing silently.
describe("availability checks", () => {
  it("asks the singular /api/user/email path", async () => {
    fetchMock.mockResolvedValue(respond(null, false, 404));

    await checkEmailAvailable("alice@example.com");

    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toContain("/api/user/email/");
    expect(url).not.toContain("/api/users/");
  });

  it("asks the singular /api/user/name path", async () => {
    fetchMock.mockResolvedValue(respond(null, false, 404));

    await checkUsernameAvailable("alice");

    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toContain("/api/user/name/");
    expect(url).not.toContain("/api/users/");
  });

  it("reports an email as available only on 404", async () => {
    fetchMock.mockResolvedValue(respond(null, false, 404));
    expect(await checkEmailAvailable("free@example.com")).toBe(true);

    fetchMock.mockResolvedValue(respond({ id: "user-1" }, true, 200));
    expect(await checkEmailAvailable("alice@example.com")).toBe(false);
  });

  it("reports a username as available only on 404", async () => {
    fetchMock.mockResolvedValue(respond(null, false, 404));
    expect(await checkUsernameAvailable("bob")).toBe(true);

    fetchMock.mockResolvedValue(respond({ id: "user-1" }, true, 200));
    expect(await checkUsernameAvailable("alice")).toBe(false);
  });

  // A 500 is not a free name - treating any non-200 as available would let
  // a duplicate through whenever the database is down.
  it("does not treat a server error as available", async () => {
    fetchMock.mockResolvedValue(respond(null, false, 500));

    expect(await checkEmailAvailable("alice@example.com")).toBe(false);
    expect(await checkUsernameAvailable("alice")).toBe(false);
  });

  it("url-encodes the value", async () => {
    fetchMock.mockResolvedValue(respond(null, false, 404));

    await checkEmailAvailable("a+b@example.com");

    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toContain(encodeURIComponent("a+b@example.com"));
  });
});
