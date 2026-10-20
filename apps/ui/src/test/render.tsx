import type { ReactElement, ReactNode } from "react";
import { vi } from "vitest";
import { render, type RenderOptions } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { AuthProvider } from "../stores/AuthContext";
import { AuthModalProvider } from "../stores/AuthModalContext";

interface ProviderOptions extends Omit<RenderOptions, "wrapper"> {
  /** Initial history entry, for components that read the location. */
  route?: string;
}

/**
 * Renders a component inside the providers the real app supplies.
 *
 * Anything calling useAuth throws outside AuthProvider, and anything
 * rendering a <Link> or calling useNavigate throws outside a router - which
 * is why rendering App or Navbar bare does not work. The router sits
 * outermost because AuthProvider doesn't depend on it, but Link does.
 */
export function renderWithProviders(
  ui: ReactElement,
  { route = "/", ...options }: ProviderOptions = {},
) {
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter initialEntries={[route]}>
        <AuthProvider>
          <AuthModalProvider>{children}</AuthModalProvider>
        </AuthProvider>
      </MemoryRouter>
    );
  }

  return render(ui, { wrapper: Wrapper, ...options });
}

/** Makes the /api/me probe resolve to a signed-in user. */
export function stubSignedInUser(user: {
  id: string;
  email: string;
  username: string;
}) {
  return vi.fn(async () => ({
    ok: true,
    status: 200,
    json: async () => user,
  }));
}
