import { act, renderHook } from "@testing-library/react";
import { vi } from "vitest";
import { AuthModalProvider, useAuthModal } from "./AuthModalContext";

function renderModal() {
  return renderHook(() => useAuthModal(), { wrapper: AuthModalProvider });
}

describe("AuthModalProvider", () => {
  it("starts closed with no pending action", () => {
    const { result } = renderModal();

    expect(result.current.isOpen).toBe(false);
    expect(result.current.pendingAction).toBeNull();
  });

  it("opens with no action when none is given", () => {
    const { result } = renderModal();

    act(() => result.current.openModal());

    expect(result.current.isOpen).toBe(true);
    expect(result.current.pendingAction).toBeNull();
  });

  // The function has to be stored as a value. Passing it straight to
  // setState would make React call it as an updater and store its return.
  it("stores a queued action as a callable, not as a state updater", () => {
    const { result } = renderModal();
    const action = vi.fn();

    act(() => result.current.openModal(action));

    expect(result.current.isOpen).toBe(true);
    expect(result.current.pendingAction).toBe(action);
    expect(action).not.toHaveBeenCalled(); // queued, not run

    result.current.pendingAction?.();
    expect(action).toHaveBeenCalledTimes(1);
  });

  it("closing discards both the open flag and the pending action", () => {
    const { result } = renderModal();
    const action = vi.fn();
    act(() => result.current.openModal(action));

    act(() => result.current.closeModal());

    expect(result.current.isOpen).toBe(false);
    expect(result.current.pendingAction).toBeNull();
    expect(action).not.toHaveBeenCalled();
  });

  it("a second open replaces the first queued action", () => {
    const { result } = renderModal();
    const first = vi.fn();
    const second = vi.fn();

    act(() => result.current.openModal(first));
    act(() => result.current.openModal(second));

    expect(result.current.pendingAction).toBe(second);
  });

  it("throws when used outside its provider", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

    expect(() => renderHook(() => useAuthModal())).toThrow(
      /must be used within an AuthModalProvider/,
    );

    consoleError.mockRestore();
  });
});
