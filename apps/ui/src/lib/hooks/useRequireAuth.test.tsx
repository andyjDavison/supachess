import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { useRequireAuth } from "./useRequireAuth";
import { useAuthModal } from "../../stores/AuthModalContext";
import { renderWithProviders } from "../../test/render";

vi.mock("../ws/client", () => ({
  gameSocket: { connect: vi.fn(), disconnect: vi.fn(), subscribe: vi.fn(() => () => {}), send: vi.fn() },
}));

const user = { id: "user-1", email: "alice@example.com", username: "alice" };

/** A button whose click is wrapped in requireAuth, plus modal state. */
function Harness({ action }: { action: () => void }) {
  const requireAuth = useRequireAuth();
  const { isOpen, pendingAction } = useAuthModal();

  return (
    <div>
      <button onClick={() => requireAuth(action)}>Do the thing</button>
      <span data-testid="modal-open">{String(isOpen)}</span>
      <button onClick={() => pendingAction?.()}>Run queued</button>
    </div>
  );
}

describe("useRequireAuth", () => {
  it("runs the action immediately when signed in", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({ ok: true, status: 200, json: async () => user })));
    const action = vi.fn();
    renderWithProviders(<Harness action={action} />);
    await waitFor(() => expect(screen.getByTestId("modal-open")).toHaveTextContent("false"));

    await userEvent.click(screen.getByText("Do the thing"));

    expect(action).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("modal-open")).toHaveTextContent("false");
  });

  it("opens the modal with the action queued when signed out", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({ ok: false, status: 401, json: async () => null })));
    const action = vi.fn();
    renderWithProviders(<Harness action={action} />);
    await waitFor(() => expect(screen.getByTestId("modal-open")).toHaveTextContent("false"));

    await userEvent.click(screen.getByText("Do the thing"));

    expect(action).not.toHaveBeenCalled();
    expect(screen.getByTestId("modal-open")).toHaveTextContent("true");

    // The queued action is what runs after a successful login.
    await userEvent.click(screen.getByText("Run queued"));
    expect(action).toHaveBeenCalledTimes(1);
  });

  // Mid-probe looks identical to logged-out from the hook's point of view,
  // so a naive guard would pop the modal at an already-signed-in user.
  it("ignores the click while the auth probe is still in flight", async () => {
    // Never settles, so isLoading stays true for the whole test.
    vi.stubGlobal("fetch", vi.fn(() => new Promise(() => {})));
    const action = vi.fn();
    renderWithProviders(<Harness action={action} />);

    await userEvent.click(screen.getByText("Do the thing"));

    expect(action).not.toHaveBeenCalled();
    expect(screen.getByTestId("modal-open")).toHaveTextContent("false");
  });
});
