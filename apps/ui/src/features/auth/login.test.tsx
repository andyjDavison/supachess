import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, vi } from "vitest";
import Login from "./login";
import { renderWithProviders } from "../../test/render";

vi.mock("../../lib/ws/client", () => ({
  gameSocket: { connect: vi.fn(), disconnect: vi.fn(), subscribe: vi.fn(() => () => {}), send: vi.fn() },
}));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => navigate,
}));

const user = { id: "user-1", email: "alice@example.com", username: "alice" };

function respond(body: unknown, ok = true, status = 200) {
  return { ok, status, json: async () => body };
}

beforeEach(() => {
  navigate.mockClear();
});

/**
 * Stubs fetch with a fixed sequence of responses, repeating the last one
 * once exhausted. The first entry answers AuthProvider's /api/me probe;
 * the rest are whatever the test triggers.
 */
function stubFetchSequence(...responses: ReturnType<typeof respond>[]) {
  const queue = [respond(null, false, 401), ...responses];
  let i = 0;
  const fetchMock = vi.fn(async () => queue[Math.min(i++, queue.length - 1)]);
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

/** Signed out, and any login attempt is rejected. */
function stubAuth() {
  return stubFetchSequence();
}

async function fillAndSubmit(email: string, password: string) {
  await userEvent.type(screen.getByPlaceholderText("Email"), email);
  await userEvent.type(screen.getByPlaceholderText("********"), password);
  await userEvent.click(screen.getByRole("button", { name: "Login" }));
}

describe("Login", () => {
  it("masks the password field", async () => {
    stubAuth();
    renderWithProviders(<Login />);

    // This was type="text", so the password was visible as typed.
    expect(screen.getByPlaceholderText("********")).toHaveAttribute("type", "password");
  });

  it("shows validation messages for an empty submit", async () => {
    stubAuth();
    renderWithProviders(<Login />);

    await userEvent.click(screen.getByRole("button", { name: "Login" }));

    await waitFor(() => {
      expect(screen.getByText("Enter a valid email address")).toBeInTheDocument();
    });
    expect(navigate).not.toHaveBeenCalled();
  });

  it("rejects a malformed email without calling the server", async () => {
    const fetchMock = stubAuth();
    renderWithProviders(<Login />);
    const callsBefore = fetchMock.mock.calls.length;

    await fillAndSubmit("not-an-email", "password123");

    await waitFor(() => {
      expect(screen.getByText("Enter a valid email address")).toBeInTheDocument();
    });
    expect(fetchMock.mock.calls.length).toBe(callsBefore);
  });

  it("navigates home after a successful login", async () => {
    // POST /api/login succeeds, then the re-probe returns the user.
    stubFetchSequence(respond(null), respond(user));
    renderWithProviders(<Login />);

    await fillAndSubmit(user.email, "password123");

    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/"));
  });

  // The handler set a root error that nothing rendered, so a wrong password
  // produced no visible feedback at all.
  it("shows the root error when the credentials are rejected", async () => {
    stubFetchSequence(respond(null, false, 401));
    renderWithProviders(<Login />);

    await fillAndSubmit(user.email, "wrong-password");

    await waitFor(() => {
      expect(
        screen.getByText("Incorrect username or password. Please try again."),
      ).toBeInTheDocument();
    });
    expect(navigate).not.toHaveBeenCalled();
  });

  it("marks the invalid field with aria-invalid", async () => {
    stubAuth();
    renderWithProviders(<Login />);

    await userEvent.click(screen.getByRole("button", { name: "Login" }));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Email")).toHaveAttribute("aria-invalid", "true");
    });
  });
});
