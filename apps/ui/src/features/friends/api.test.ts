import { afterEach, describe, expect, it, vi } from "vitest";
import {
  FriendsApiError,
  acceptFriendRequest,
  blockUser,
  deleteFriendRequest,
  listFriends,
  listRequests,
  searchUsers,
  sendFriendRequest,
} from "./api";

function stubFetch(response: { ok?: boolean; status?: number; json?: unknown; text?: string }) {
  const fetchMock = vi.fn(async () => ({
    ok: response.ok ?? true,
    status: response.status ?? 200,
    json: async () => response.json,
    text: async () => response.text ?? "",
  }));
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function lastCall(fetchMock: ReturnType<typeof stubFetch>) {
  const [url, init] = fetchMock.mock.calls.at(-1) as unknown as [string, RequestInit];
  return { url, init };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("friends api", () => {
  it("sends the session cookie on every request", async () => {
    const fetchMock = stubFetch({ json: { friends: [] } });

    await listFriends();

    expect(lastCall(fetchMock).init.credentials).toBe("include");
  });

  it("unwraps and validates the friends list", async () => {
    stubFetch({
      json: { friends: [{ userId: "u2", username: "bob", rating: 1200, sinceMs: 1_700_000_000_000 }] },
    });

    await expect(listFriends()).resolves.toEqual([
      { userId: "u2", username: "bob", rating: 1200, sinceMs: 1_700_000_000_000 },
    ]);
  });

  it("rejects a response that doesn't match the contract", async () => {
    stubFetch({ json: { incoming: null, outgoing: [] } });

    await expect(listRequests()).rejects.toThrow();
  });

  it("posts the target id and returns the resulting status", async () => {
    const fetchMock = stubFetch({ status: 201, json: { status: "accepted" } });

    await expect(sendFriendRequest("u2")).resolves.toBe("accepted");

    const { url, init } = lastCall(fetchMock);
    expect(url).toMatch(/\/api\/friends\/requests$/);
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual({ userId: "u2" });
  });

  it.each([
    ["accept", () => acceptFriendRequest("u 2"), "POST", "/api/friends/requests/u%202/accept"],
    ["decline/cancel", () => deleteFriendRequest("u2"), "DELETE", "/api/friends/requests/u2"],
    ["block", () => blockUser("u2"), "PUT", "/api/friends/blocks/u2"],
  ])("%s hits the right route", async (_name, call, method, path) => {
    const fetchMock = stubFetch({ status: 204 });

    await call();

    const { url, init } = lastCall(fetchMock);
    expect(init.method).toBe(method);
    expect(url.endsWith(path)).toBe(true);
  });

  it("encodes the search query", async () => {
    const fetchMock = stubFetch({ json: { results: [] } });

    await searchUsers("a&b");

    expect(lastCall(fetchMock).url).toMatch(/\/api\/friends\/search\?q=a%26b$/);
  });

  it("throws the server's message with its status", async () => {
    stubFetch({ ok: false, status: 409, text: "friend request already exists\n" });

    const err = await sendFriendRequest("u2").catch((e: unknown) => e);

    expect(err).toBeInstanceOf(FriendsApiError);
    expect(err).toMatchObject({ status: 409, message: "friend request already exists" });
  });
});
