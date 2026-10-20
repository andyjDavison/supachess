import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { AuthModal } from "./auth-modal";
import { useAuthModal } from "../../stores/AuthModalContext";
import { renderWithProviders } from "../../test/render";

vi.mock("../../lib/ws/client", () => ({
  gameSocket: { connect: vi.fn(), disconnect: vi.fn(), subscribe: vi.fn(() => () => {}), send: vi.fn() },
}));

const user = { id: "user-1", email: "alice@example.com", username: "alice" };

function respond(body: unknown, ok = true, status = 200) {
  return { ok, status, json: async () => body };
}

function stubFetchSequence(...responses: ReturnType<typeof respond>[]) {
  const queue = [respond(null, false, 401), ...responses];
  let i = 0;
  vi.stubGlobal("fetch", vi.fn(async () => queue[Math.min(i++, queue.length - 1)]));
}

/** Renders the modal plus a trigger that opens it with a queued action. */
function Harness({ action }: { action?: () => void }) {
  const { openModal } = useAuthModal();
  return (
    <>
      <button onClick={() => openModal(action)}>Open it</button>
      <AuthModal />
    </>
  );
}

async function open(action?: () => void) {
  renderWithProviders(<Harness action={action} />);
  await userEvent.click(screen.getByRole("button", { name: "Open it" }));
}

async function submitCredentials(password = "password123") {
  await userEvent.type(screen.getByLabelText("Email"), user.email);
  await userEvent.type(screen.getByLabelText("Password"), password);
  await userEvent.click(screen.getByRole("button", { name: "Log in" }));
}

describe("AuthModal", () => {
  it("renders nothing until it is opened", () => {
    stubFetchSequence();
    renderWithProviders(<Harness />);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("appears when opened", async () => {
    stubFetchSequence();
    await open();

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Log in to continue")).toBeInTheDocument();
  });

  it("closes on Escape", async () => {
    stubFetchSequence();
    await open();

    await userEvent.keyboard("{Escape}");

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("closes when the backdrop is clicked", async () => {
    stubFetchSequence();
    await open();
    const dialog = screen.getByRole("dialog");

    // The backdrop is the dialog's parent; clicking the panel itself must not close.
    await userEvent.click(dialog.parentElement as HTMLElement);

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("stays open when the panel itself is clicked", async () => {
    stubFetchSequence();
    await open();

    await userEvent.click(screen.getByText("Log in to continue"));

    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("closes on Cancel", async () => {
    stubFetchSequence();
    await open();

    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("validates before submitting", async () => {
    stubFetchSequence();
    await open();

    await userEvent.click(screen.getByRole("button", { name: "Log in" }));

    await waitFor(() => {
      expect(screen.getByText("Enter a valid email address")).toBeInTheDocument();
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("shows an error and stays open when the credentials are wrong", async () => {
    stubFetchSequence(respond(null, false, 401));
    await open();

    await submitCredentials("wrong");

    await waitFor(() => {
      expect(screen.getByText("Invalid email or password")).toBeInTheDocument();
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  // The whole point of the modal: the action the user originally clicked
  // runs after they sign in, without navigating them away.
  it("runs the queued action after a successful login, then closes", async () => {
    stubFetchSequence(respond(null), respond(user));
    const action = vi.fn();
    await open(action);

    await submitCredentials();

    await waitFor(() => expect(action).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("closes cleanly when there is no queued action", async () => {
    stubFetchSequence(respond(null), respond(user));
    await open();

    await submitCredentials();

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("masks the password field", async () => {
    stubFetchSequence();
    await open();

    expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
  });
});
