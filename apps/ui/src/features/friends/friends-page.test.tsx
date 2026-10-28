import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router";
import FriendsPage from "./friends-page";

// The Message button is a router Link.
const renderPage = () => render(<FriendsPage />, { wrapper: MemoryRouter });

type Route = { method: string; path: RegExp; status?: number; body?: unknown };

/**
 * A tiny fake backend: each call is matched against `routes` by method and
 * path, first match wins. Tests mutate `state` to change what the next
 * load returns, the way the real server would after an action.
 */
function stubApi(routes: () => Route[]) {
  const fetchMock = vi.fn(async (url: string, init: RequestInit = {}) => {
    const method = init.method ?? "GET";
    const route = routes().find((r) => r.method === method && r.path.test(url));
    if (!route) throw new Error(`unexpected ${method} ${url}`);
    const status = route.status ?? 200;
    return {
      ok: status < 400,
      status,
      json: async () => route.body,
      text: async () => (typeof route.body === "string" ? route.body : ""),
    };
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

const bob = { userId: "u-bob", username: "bob", rating: 1300 };
const carol = { userId: "u-carol", username: "carol", rating: 1100 };

let state: {
  friends: unknown[];
  incoming: unknown[];
  outgoing: unknown[];
  blocked: unknown[];
  results: unknown[];
};

function listRoutes(): Route[] {
  return [
    { method: "GET", path: /\/api\/friends$/, body: { friends: state.friends } },
    { method: "GET", path: /\/api\/friends\/requests$/, body: { incoming: state.incoming, outgoing: state.outgoing } },
    { method: "GET", path: /\/api\/friends\/blocks$/, body: { blocked: state.blocked } },
    { method: "GET", path: /\/api\/friends\/search\?q=/, body: { results: state.results } },
  ];
}

beforeEach(() => {
  state = { friends: [], incoming: [], outgoing: [], blocked: [], results: [] };
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("FriendsPage", () => {
  it("lists friends and incoming requests", async () => {
    state.friends = [{ ...bob, sinceMs: 1 }];
    state.incoming = [{ ...carol, createdAtMs: 1 }];
    stubApi(listRoutes);

    renderPage();

    const friends = await screen.findByRole("region", { name: "Friends (1)" });
    expect(within(friends).getByText("bob")).toBeInTheDocument();
    const requests = screen.getByRole("region", { name: "Requests (1)" });
    expect(within(requests).getByText("carol")).toBeInTheDocument();
  });

  it("shows an empty state with no friends", async () => {
    stubApi(listRoutes);

    renderPage();

    expect(await screen.findByText(/No friends yet/)).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: /Requests/ })).not.toBeInTheDocument();
  });

  it("accepting a request calls the api and reloads the lists", async () => {
    state.incoming = [{ ...carol, createdAtMs: 1 }];
    const fetchMock = stubApi(() => [
      {
        method: "POST",
        path: /\/api\/friends\/requests\/u-carol\/accept$/,
        status: 204,
      },
      ...listRoutes(),
    ]);
    renderPage();
    const requests = await screen.findByRole("region", { name: "Requests (1)" });

    // What the server returns once the accept has landed.
    state.incoming = [];
    state.friends = [{ ...carol, sinceMs: 2 }];
    await userEvent.click(within(requests).getByRole("button", { name: "Accept" }));

    const friends = await screen.findByRole("region", { name: "Friends (1)" });
    expect(within(friends).getByText("carol")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: /Requests/ })).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringMatching(/\/accept$/),
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
  });

  it("shows the button that fits each search result's relationship", async () => {
    state.results = [
      { ...bob, relationship: "none" },
      { ...carol, relationship: "outgoing" },
      { userId: "u-dan", username: "dan", rating: 900, relationship: "friend" },
      { userId: "u-eve", username: "eve", rating: 1000, relationship: "incoming" },
    ];
    stubApi(listRoutes);
    renderPage();

    await userEvent.type(screen.getByLabelText("Search by username"), "a");

    const search = screen.getByRole("region", { name: "Find players" });
    const row = async (name: string) => (await within(search).findByText(name)).closest("div")!.parentElement!;
    expect(within(await row("bob")).getByRole("button", { name: "Add friend" })).toBeInTheDocument();
    expect(within(await row("carol")).getByText("Pending")).toBeInTheDocument();
    expect(within(await row("dan")).getByText("Friends")).toBeInTheDocument();
    expect(within(await row("eve")).getByRole("button", { name: "Accept" })).toBeInTheDocument();
  });

  it("sending a request posts the user id and refreshes the search result", async () => {
    state.results = [{ ...bob, relationship: "none" }];
    const fetchMock = stubApi(() => [
      { method: "POST", path: /\/api\/friends\/requests$/, status: 201, body: { status: "pending" } },
      ...listRoutes(),
    ]);
    renderPage();
    await userEvent.type(screen.getByLabelText("Search by username"), "bo");
    const add = await screen.findByRole("button", { name: "Add friend" });

    state.results = [{ ...bob, relationship: "outgoing" }];
    state.outgoing = [{ ...bob, createdAtMs: 1 }];
    await userEvent.click(add);

    const search = screen.getByRole("region", { name: "Find players" });
    expect(await within(search).findByText("Pending")).toBeInTheDocument();
    expect(await screen.findByRole("region", { name: "Sent (1)" })).toBeInTheDocument();
    const post = fetchMock.mock.calls.find(([, init]) => init?.method === "POST")!;
    expect(JSON.parse(post[1]!.body as string)).toEqual({ userId: "u-bob" });
  });

  it("blocks only after confirming", async () => {
    state.friends = [{ ...bob, sinceMs: 1 }];
    const fetchMock = stubApi(() => [
      { method: "PUT", path: /\/api\/friends\/blocks\/u-bob$/, status: 204 },
      ...listRoutes(),
    ]);
    const confirm = vi.spyOn(window, "confirm").mockReturnValueOnce(false).mockReturnValueOnce(true);
    renderPage();
    const friends = await screen.findByRole("region", { name: "Friends (1)" });

    await userEvent.click(within(friends).getByRole("button", { name: "Block" }));
    expect(fetchMock).not.toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ method: "PUT" }));

    await userEvent.click(within(friends).getByRole("button", { name: "Block" }));
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringMatching(/\/blocks\/u-bob$/),
        expect.objectContaining({ method: "PUT" }),
      ),
    );
    expect(confirm).toHaveBeenCalledTimes(2);
  });

  it("surfaces a failed action's server message", async () => {
    state.results = [{ ...bob, relationship: "none" }];
    stubApi(() => [
      { method: "POST", path: /\/api\/friends\/requests$/, status: 404, body: "not found\n" },
      ...listRoutes(),
    ]);
    renderPage();
    await userEvent.type(screen.getByLabelText("Search by username"), "bo");

    await userEvent.click(await screen.findByRole("button", { name: "Add friend" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("not found");
  });

  it("lists blocked users behind a toggle and can unblock", async () => {
    state.blocked = [{ userId: "u-bob", username: "bob", blockedAtMs: 1 }];
    const fetchMock = stubApi(() => [
      { method: "DELETE", path: /\/api\/friends\/blocks\/u-bob$/, status: 204 },
      ...listRoutes(),
    ]);
    renderPage();

    const toggle = await screen.findByRole("button", { name: /Blocked \(1\)/ });
    expect(screen.queryByText("bob")).not.toBeInTheDocument();
    await userEvent.click(toggle);

    state.blocked = [];
    await userEvent.click(screen.getByRole("button", { name: "Unblock" }));

    await waitFor(() => expect(screen.queryByRole("button", { name: /Blocked/ })).not.toBeInTheDocument());
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringMatching(/\/blocks\/u-bob$/),
      expect.objectContaining({ method: "DELETE" }),
    );
  });
});
