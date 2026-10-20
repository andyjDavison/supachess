import { screen, waitFor } from "@testing-library/react";
import App from "./App";
import { renderWithProviders } from "./test/render";

describe("App", () => {
  it("renders its routed outlet without crashing", async () => {
    renderWithProviders(<App />);

    // App calls useAuth, so it needs AuthProvider; the /api/me probe
    // resolving is what settles the initial render.
    await waitFor(() => {
      expect(screen.getByRole("main")).toBeInTheDocument();
    });
  });
});
