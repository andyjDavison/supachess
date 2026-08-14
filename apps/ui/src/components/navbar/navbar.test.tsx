import { render, screen } from "@testing-library/react";
import Navbar from "./navbar";

describe("Navbar", () => {
  it("should render the play button", () => {
    render(<Navbar />);

    expect(screen.getByRole("button", { name: "Play" }));
  });
});
