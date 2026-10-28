import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test/render";
import type { Game } from "../../../lib/ws/schemas";
import AnalysisSidePanel from "./analysis";

vi.mock("../../game/move-list", () => ({ MoveList: () => <p>move list</p> }));
vi.mock("../../messages/game-chat", () => ({ GameChat: ({ game }: { game: Game }) => <p>chat for {game.id}</p> }));

const game = { id: "game-1" } as Game;

describe("AnalysisSidePanel", () => {
  it("shows Moves first, with Chat as a sub-tab beside it", async () => {
    renderWithProviders(<AnalysisSidePanel game={game} />);

    expect(screen.getByRole("tab", { name: "Moves" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("move list")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("tab", { name: "Chat" }));

    expect(screen.getByText("chat for game-1")).toBeInTheDocument();
    expect(screen.queryByText("move list")).not.toBeInTheDocument();
  });
});
