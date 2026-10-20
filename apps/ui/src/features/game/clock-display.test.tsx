import { render, screen } from "@testing-library/react";
import { ClockDisplay } from "./clock-display";

describe("ClockDisplay", () => {
  it("renders the formatted remaining time", () => {
    render(<ClockDisplay isWhite ms={185_000} isActive={false} />);

    expect(screen.getByText("3:05")).toBeInTheDocument();
  });

  it("marks time low once under thirty seconds, and only while running", () => {
    const { rerender, container } = render(
      <ClockDisplay isWhite ms={29_000} isActive />,
    );
    const classesWhileLow = container.firstElementChild?.className ?? "";

    expect(classesWhileLow).toContain("text-red-400");

    // Not the player's turn: the clock isn't running, so no warning.
    rerender(<ClockDisplay isWhite ms={29_000} isActive={false} />);
    expect(container.firstElementChild?.className ?? "").not.toContain(
      "text-red-400",
    );

    // Running but plenty of time left.
    rerender(<ClockDisplay isWhite ms={60_000} isActive />);
    expect(container.firstElementChild?.className ?? "").not.toContain(
      "text-red-400",
    );
  });

  // Exactly one state colour should apply. An earlier version set
  // text-clock-white-d/-black-d and then text-clock-active/-disabled on
  // top, and the second pair had no tokens behind it.
  it("applies a single state colour", () => {
    const { container, rerender } = render(
      <ClockDisplay isWhite ms={60_000} isActive />,
    );

    const active = container.firstElementChild?.className ?? "";
    expect(active).toContain("text-clock-active");
    expect(active).not.toContain("text-clock-disabled");

    rerender(<ClockDisplay isWhite ms={60_000} isActive={false} />);

    const idle = container.firstElementChild?.className ?? "";
    expect(idle).toContain("text-clock-disabled");
    expect(idle).not.toContain("text-clock-active");
  });

  it("gives white's idle clock the light ground and everything else the dark one", () => {
    const { container, rerender } = render(
      <ClockDisplay isWhite ms={60_000} isActive={false} />,
    );
    expect(container.firstElementChild?.className ?? "").toContain("bg-clock-bg");

    rerender(<ClockDisplay isWhite ms={60_000} isActive />);
    expect(container.firstElementChild?.className ?? "").toContain("bg-nav-bg");

    rerender(<ClockDisplay isWhite={false} ms={60_000} isActive={false} />);
    expect(container.firstElementChild?.className ?? "").toContain("bg-nav-bg");
  });

  it("shows a flagged clock as 0:00", () => {
    render(<ClockDisplay isWhite ms={0} isActive />);

    expect(screen.getByText("0:00")).toBeInTheDocument();
  });
});
