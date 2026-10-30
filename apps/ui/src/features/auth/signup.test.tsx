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

const next = () => userEvent.click(screen.getByRole("button", { name: "Continue" }));

async function chooseSkill(text = "I know the rules and basics") {
  await userEvent.click(screen.getByText(text));
  await next();
}

async function enterCredentials(email = "alice@example.com", password = "password123") {
  await userEvent.type(screen.getByLabelText("Email"), email);
  await userEvent.type(screen.getByLabelText("Password"), password);
  await next();
}

async function finish(username = "alice") {
  await userEvent.type(await screen.findByLabelText("Username"), username);
  await userEvent.click(screen.getByRole("button", { name: "Create Account" }));
}

describe("Signup", () => {
  it("starts on the skill step, and won't continue without a choice", async () => {
    renderWithProviders(<Signup />);

    expect(screen.getByRole("list", { name: "Step 1 of 3" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue" })).toBeDisabled();

    await userEvent.click(screen.getByText("I'm a tournament player"));
    expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled();
  });

  // The four skill levels are one choice, not four independent checkboxes.
  it("keeps the skill levels mutually exclusive", async () => {
    renderWithProviders(<Signup />);

    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(4);

    await userEvent.click(screen.getByText("I don't know how to play"));
    expect(radios.filter((r) => (r as HTMLInputElement).checked)).toHaveLength(1);

    await userEvent.click(screen.getByText("I'm a tournament player"));
    const checked = radios.filter((r) => (r as HTMLInputElement).checked);
    expect(checked).toHaveLength(1);
    expect((checked[0] as HTMLInputElement).value).toBe("1600");
  });

  it("asks for email and password second, masking the password", async () => {
    renderWithProviders(<Signup />);
    await chooseSkill();

    expect(screen.getByRole("list", { name: "Step 2 of 3" })).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
    for (const provider of ["Continue with Google", "Continue with Apple"]) {
      expect(screen.getByRole("button", { name: provider })).toBeDisabled();
    }
  });

  it("won't leave step 2 with an invalid email or a short password", async () => {
    renderWithProviders(<Signup />);
    await chooseSkill();

    await enterCredentials("not-an-email", "short");

    expect(await screen.findByText("Enter a valid email address")).toBeInTheDocument();
    expect(screen.getByText("Password must be at least 8 characters")).toBeInTheDocument();
    expect(screen.queryByLabelText("Username")).not.toBeInTheDocument();
  });

  it("shows a message for a username under three characters", async () => {
    renderWithProviders(<Signup />);
    await chooseSkill();
    await enterCredentials();

    await finish("ab");

    expect(await screen.findByText("Username must be at least 3 characters")).toBeInTheDocument();
    expect(registerUser).not.toHaveBeenCalled();
  });

  it("submits the registration and navigates home", async () => {
    renderWithProviders(<Signup />);
    await chooseSkill();
    await enterCredentials();
    await finish();

    await waitFor(() => expect(registerUser).toHaveBeenCalledTimes(1));
    expect(vi.mocked(registerUser).mock.calls[0][0]).toEqual({
      email: "alice@example.com",
      password: "password123",
      username: "alice",
      rating: 800,
    });
    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/"));
  });

  it("Back keeps what you entered", async () => {
    renderWithProviders(<Signup />);
    await chooseSkill("I'm a tournament player");
    await enterCredentials();

    await userEvent.click(screen.getByRole("button", { name: "← Back" }));
    expect(screen.getByLabelText("Email")).toHaveValue("alice@example.com");

    await userEvent.click(screen.getByRole("button", { name: "← Back" }));
    expect(screen.getByRole("radio", { checked: true })).toHaveAttribute("value", "1600");
  });

  it("a taken username stays on the username step with a message", async () => {
    vi.mocked(registerUser).mockRejectedValue(new Error("username already taken"));
    renderWithProviders(<Signup />);
    await chooseSkill();
    await enterCredentials();
    await finish();

    expect(await screen.findByText("That username is taken.")).toBeInTheDocument();
    expect(navigate).not.toHaveBeenCalled();
  });

  it("a taken email goes back to the email step", async () => {
    vi.mocked(registerUser).mockRejectedValue(new Error("email already registered"));
    renderWithProviders(<Signup />);
    await chooseSkill();
    await enterCredentials();
    await finish();

    expect(await screen.findByText("That email is already registered.")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toHaveValue("alice@example.com");
  });
});
