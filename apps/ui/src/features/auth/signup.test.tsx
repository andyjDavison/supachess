import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, vi } from "vitest";
import Signup from "./signup";
import { renderWithProviders } from "../../test/render";
import { registerUser } from "../../lib/services/user";

vi.mock("../../lib/services/user", () => ({ registerUser: vi.fn() }));

const navigate = vi.fn();
vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => navigate,
}));

beforeEach(() => {
  navigate.mockClear();
  vi.mocked(registerUser).mockReset();
  vi.mocked(registerUser).mockResolvedValue({ id: "user-1", username: "alice", rating: 800 });
});

/** Clicks through the provider-choice step to the email form. */
async function openEmailForm() {
  renderWithProviders(<Signup />);
  await userEvent.click(screen.getByRole("button", { name: "Continue with Email" }));
}

describe("the provider step", () => {
  it("offers email sign-up and disables the unimplemented providers", () => {
    renderWithProviders(<Signup />);

    expect(screen.getByRole("button", { name: "Continue with Email" })).toBeEnabled();
    for (const provider of ["Continue with Phone", "Continue with Google", "Continue with Apple"]) {
      expect(screen.getByRole("button", { name: provider })).toBeDisabled();
    }
  });

  it("shows the form only after choosing email", async () => {
    renderWithProviders(<Signup />);

    expect(screen.queryByPlaceholderText("Username")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Continue with Email" }));

    expect(screen.getByPlaceholderText("Username")).toBeInTheDocument();
  });
});

describe("the email form", () => {
  it("masks the password field", async () => {
    await openEmailForm();

    // This was type="text", so the password was visible as typed.
    expect(screen.getByPlaceholderText("********")).toHaveAttribute("type", "password");
  });

  it("does not submit without a skill level chosen", async () => {
    await openEmailForm();

    await userEvent.type(screen.getByPlaceholderText("Email"), "alice@example.com");
    await userEvent.type(screen.getByPlaceholderText("********"), "password123");
    await userEvent.type(screen.getByPlaceholderText("Username"), "alice");
    await userEvent.click(screen.getByRole("button", { name: "Register" }));

    await waitFor(() => expect(registerUser).not.toHaveBeenCalled());
  });

  it("shows a message for a password under eight characters", async () => {
    await openEmailForm();

    await userEvent.type(screen.getByPlaceholderText("********"), "short");
    await userEvent.tab(); // schema validates onBlur

    await waitFor(() => {
      expect(screen.getByText("Password must be at least 8 characters")).toBeInTheDocument();
    });
  });

  it("shows a message for a username under three characters", async () => {
    await openEmailForm();

    await userEvent.type(screen.getByPlaceholderText("Username"), "ab");
    await userEvent.tab();

    await waitFor(() => {
      expect(screen.getByText("Username must be at least 3 characters")).toBeInTheDocument();
    });
  });

  it("submits the registration and navigates home", async () => {
    await openEmailForm();

    await userEvent.type(screen.getByPlaceholderText("Email"), "alice@example.com");
    await userEvent.type(screen.getByPlaceholderText("********"), "password123");
    await userEvent.type(screen.getByPlaceholderText("Username"), "alice");
    await userEvent.click(screen.getByText("I know the rules and basics"));
    await userEvent.click(screen.getByRole("button", { name: "Register" }));

    await waitFor(() => expect(registerUser).toHaveBeenCalledTimes(1));
    expect(vi.mocked(registerUser).mock.calls[0][0]).toEqual({
      email: "alice@example.com",
      password: "password123",
      username: "alice",
      rating: 800,
    });
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/"));
  });

  // The four skill levels are one choice, not four independent checkboxes.
  it("keeps the skill levels mutually exclusive", async () => {
    await openEmailForm();

    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(4);

    await userEvent.click(screen.getByText("I don't know how to play"));
    expect(radios.filter((r) => (r as HTMLInputElement).checked)).toHaveLength(1);

    await userEvent.click(screen.getByText("I'm a tournament player"));
    const checked = radios.filter((r) => (r as HTMLInputElement).checked);
    expect(checked).toHaveLength(1);
    expect((checked[0] as HTMLInputElement).value).toBe("1600");
  });

  it("stays on the form when registration fails", async () => {
    vi.mocked(registerUser).mockRejectedValue(new Error("username already taken"));
    await openEmailForm();

    await userEvent.type(screen.getByPlaceholderText("Email"), "alice@example.com");
    await userEvent.type(screen.getByPlaceholderText("********"), "password123");
    await userEvent.type(screen.getByPlaceholderText("Username"), "alice");
    await userEvent.click(screen.getByText("I know the rules and basics"));
    await userEvent.click(screen.getByRole("button", { name: "Register" }));

    await waitFor(() => expect(registerUser).toHaveBeenCalled());
    expect(navigate).not.toHaveBeenCalled();
    expect(screen.getByPlaceholderText("Username")).toBeInTheDocument();
  });
});
