import { screen } from "@testing-library/react";
import Navbar from "./navbar";
import { renderWithProviders } from "../../test/render";

function hrefs() {
  return screen.getAllByRole("link").map((link) => link.getAttribute("href"));
}

describe("Navbar", () => {
  it("links to every top-level section", () => {
    renderWithProviders(<Navbar />);

    for (const target of ["/play", "/puzzles", "/community"]) {
      expect(hrefs()).toContain(target);
    }
  });

  // The submenus are in the DOM and hidden with group-hover CSS rather
  // than conditionally rendered, so their links are queryable here.
  it("renders each submenu's destinations", () => {
    renderWithProviders(<Navbar />);

    for (const target of [
      "/play/online",
      "/play/computer",
      "/daily",
      "/puzzles/rush",
      "/friends",
      "/clubs",
      "/members",
    ]) {
      expect(hrefs()).toContain(target);
    }
  });

  it("shows the brand", () => {
    renderWithProviders(<Navbar />);

    expect(screen.getByText("Supachess")).toBeInTheDocument();
  });

  // Logged out is the default the fetch stub in setupTests establishes.
  it("offers sign up and log in when nobody is signed in", () => {
    renderWithProviders(<Navbar />);

    expect(hrefs()).toContain("/register");
    expect(hrefs()).toContain("/login");
  });
});
