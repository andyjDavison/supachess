import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useBoardAppearance, useBoardSettings } from "./board-settings";

const KEY = "supachess.board-settings";
const state = () => useBoardSettings.getState();

beforeEach(() => {
  localStorage.clear();
  state().reload();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("board settings", () => {
  it("defaults to the Supachess theme with coordinates", () => {
    expect(state().themeId).toBe("supachess");
    expect(state().showCoordinates).toBe(true);
  });

  it("saves changes and reads them back", () => {
    state().setTheme("green");
    state().setShowCoordinates(false);

    expect(JSON.parse(localStorage.getItem(KEY)!)).toEqual({ themeId: "green", showCoordinates: false });
    useBoardSettings.setState({ themeId: "supachess", showCoordinates: true });
    state().reload();
    expect(state().themeId).toBe("green");
    expect(state().showCoordinates).toBe(false);
  });

  it("ignores a saved theme that no longer exists", () => {
    localStorage.setItem(KEY, JSON.stringify({ themeId: "neon", showCoordinates: false }));

    state().reload();

    expect(state().themeId).toBe("supachess");
    expect(state().showCoordinates).toBe(false);
  });

  it("survives storage that throws", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("blocked");
    });

    state().reload();
    expect(state().themeId).toBe("supachess");

    state().setTheme("blue");
    expect(state().themeId).toBe("blue");
  });
});

it("the board's square colors and coordinates follow the settings", () => {
  const { result } = renderHook(() => useBoardAppearance());
  expect(result.current.darkSquareStyle).toEqual({ background: "var(--color-accent)" });

  act(() => {
    state().setTheme("green");
    state().setShowCoordinates(false);
  });

  expect(result.current).toEqual({
    lightSquareStyle: { backgroundColor: "#ebecd0" },
    darkSquareStyle: { background: "#739552" },
    showNotation: false,
  });
});
