import { formatClock } from "./format-clock";

describe("formatClock", () => {
  it("formats whole minutes", () => {
    expect(formatClock(180_000)).toBe("3:00");
    expect(formatClock(600_000)).toBe("10:00");
  });

  it("zero-pads the seconds", () => {
    expect(formatClock(61_000)).toBe("1:01");
    expect(formatClock(69_000)).toBe("1:09");
  });

  it("does not pad the minutes", () => {
    expect(formatClock(9_000)).toBe("0:09");
  });

  // Rounding up matters: a clock reading 0:00 must mean the time is
  // actually gone, so any remaining fraction of a second shows as 1.
  it("rounds up so a part-second still shows", () => {
    expect(formatClock(1)).toBe("0:01");
    expect(formatClock(999)).toBe("0:01");
    expect(formatClock(1_001)).toBe("0:02");
  });

  it("shows 0:00 only at or below zero", () => {
    expect(formatClock(0)).toBe("0:00");
  });

  it("clamps negatives rather than rendering them", () => {
    expect(formatClock(-1)).toBe("0:00");
    expect(formatClock(-5_000)).toBe("0:00");
  });

  it("handles long time controls", () => {
    expect(formatClock(3_600_000)).toBe("60:00");
    expect(formatClock(5_400_000)).toBe("90:00");
  });
});
