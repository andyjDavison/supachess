import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";
import { PlayerBar } from "./player-bar";
import { BoardSettingsButton } from "./board-settings";
import { useBoardSettings } from "../../stores/board-settings";

describe("PlayerBar", () => {
  it("shows the name, rating, captures and material lead", () => {
    render(<PlayerBar color="white" name="alice" rating={1234} captured={["p", "p", "n"]} advantage={5} />);

    const bar = screen.getByTestId("player-bar-white");
    expect(bar).toHaveTextContent("alice(1234)");
    expect(screen.getByLabelText("Captured: p p n")).toHaveTextContent("+5");
  });

  it("hides the lead when behind or level", () => {
    render(<PlayerBar color="black" name="bob" captured={["p"]} advantage={-2} />);

    expect(screen.getByLabelText("Captured: p")).not.toHaveTextContent("+");
  });

  it("shows the clock when given one", () => {
    render(<PlayerBar color="white" name="alice" clock={{ ms: 65_000, isActive: true }} />);

    expect(screen.getByText("1:05")).toBeInTheDocument();
  });
});

describe("board settings", () => {
  beforeEach(() => {
    localStorage.clear();
    useBoardSettings.getState().reload();
  });

  it("the gear opens the settings, and picking a theme applies and closes on Escape", async () => {
    render(<BoardSettingsButton />);

    await userEvent.click(screen.getByRole("button", { name: "Board settings" }));
    const dialog = screen.getByRole("dialog", { name: "Board Settings" });
    expect(screen.getByRole("radio", { name: "Supachess" })).toHaveAttribute("aria-checked", "true");

    await userEvent.click(screen.getByRole("radio", { name: "Green" }));
    expect(screen.getByRole("radio", { name: "Green" })).toHaveAttribute("aria-checked", "true");
    expect(useBoardSettings.getState().themeId).toBe("green");
    expect(dialog).toBeInTheDocument();

    await userEvent.click(screen.getByLabelText("Show coordinates"));
    expect(useBoardSettings.getState().showCoordinates).toBe(false);

    await userEvent.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
