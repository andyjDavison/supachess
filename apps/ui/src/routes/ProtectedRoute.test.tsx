import { render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router";
import { ProtectedLayout } from "./ProtectedRoute";
import { AuthProvider } from "../stores/AuthContext";

vi.mock("../lib/ws/client", () => ({
  gameSocket: { connect: vi.fn(), disconnect: vi.fn(), subscribe: vi.fn(() => () => {}), send: vi.fn() },
}));

const user = { id: "user-1", email: "alice@example.com", username: "alice" };

function renderGuarded(initialPath = "/game/1") {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <AuthProvider>
        <Routes>
          <Route element={<ProtectedLayout />}>
            <Route path="/game/:id" element={<div>the protected game</div>} />
          </Route>
          <Route path="/login" element={<div>the login page</div>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe("ProtectedLayout", () => {
  it("shows a loading state while the auth probe is in flight", () => {
    // A fetch that never settles keeps isLoading true.
    vi.stubGlobal("fetch", vi.fn(() => new Promise(() => {})));

    renderGuarded();

    expect(screen.getByText("Loading...")).toBeInTheDocument();
    expect(screen.queryByText("the protected game")).not.toBeInTheDocument();
    expect(screen.queryByText("the login page")).not.toBeInTheDocument();
  });

  // Redirecting before the probe resolves would bounce a signed-in user to
  // the login page on every hard refresh.
  it("does not redirect before the probe resolves", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise(() => {})));

    renderGuarded();

    expect(screen.queryByText("the login page")).not.toBeInTheDocument();
  });

  it("redirects to /login when nobody is signed in", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({ ok: false, status: 401, json: async () => null })));

    renderGuarded();

    await waitFor(() => expect(screen.getByText("the login page")).toBeInTheDocument());
    expect(screen.queryByText("the protected game")).not.toBeInTheDocument();
  });

  it("renders the nested route when signed in", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({ ok: true, status: 200, json: async () => user })));

    renderGuarded();

    await waitFor(() => expect(screen.getByText("the protected game")).toBeInTheDocument());
    expect(screen.queryByText("the login page")).not.toBeInTheDocument();
  });
});
