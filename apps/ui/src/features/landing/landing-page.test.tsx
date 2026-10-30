import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import LandingPage from "./landing-page";
import { renderWithProviders } from "../../test/render";

vi.mock("../../lib/ws/client", () => ({
  gameSocket: { connect: vi.fn(), disconnect: vi.fn(), subscribe: vi.fn(() => () => {}), send: vi.fn() },
}));

function stubMe(user: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => (user ? { ok: true, status: 200, json: async () => user } : { ok: false, status: 401, json: async () => null })),
  );
}

afterEach(() => vi.unstubAllGlobals());

describe("LandingPage", () => {
  it("sends a visitor to sign up, with a log-in link", async () => {
    stubMe(null);
    renderWithProviders(<LandingPage />);

    expect(await screen.findByRole("link", { name: "Get Started" })).toHaveAttribute("href", "/register");
    expect(screen.getByRole("link", { name: "Log In" })).toHaveAttribute("href", "/login");
    expect(screen.getByRole("link", { name: "Play Bots" })).toHaveAttribute("href", "/play/computer");
  });

  it("sends a signed-in player straight to a game", async () => {
    stubMe({ id: "u1", email: "a@example.com", username: "alice" });
    renderWithProviders(<LandingPage />);

    expect(await screen.findByRole("link", { name: "Play Online" })).toHaveAttribute("href", "/play/online");
    await waitFor(() => expect(screen.queryByRole("link", { name: "Log In" })).not.toBeInTheDocument());
  });

  it("links the feature tiles", async () => {
    stubMe(null);
    renderWithProviders(<LandingPage />);

    expect(await screen.findByRole("link", { name: /Solve Puzzles/ })).toHaveAttribute("href", "/puzzles");
    expect(screen.getByRole("link", { name: /Join the Community/ })).toHaveAttribute("href", "/community");
  });
});
