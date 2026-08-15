import NavbarButton from "./navbar-button";

function Navbar() {
  return (
    <nav className="flex flex-col fixed top-0 left-0 w-38 h-screen bg-nav-bg z-50">
      <h2 className="flex items-center pl-7 h-14">Supachess</h2>
      <NavbarButton
        title="Play"
        link="play"
        options={[
          { title: "Play Online", link: "play/online" },
          { title: "Play Bots", link: "play/computer" },
        ]}
      />
      <NavbarButton
        title="Puzzles"
        link="puzzles"
        options={[
          { title: "Puzzles", link: "puzzles" },
          { title: "Daily Puzzles", link: "daily" },
          { title: "Puzzle Rush", link: "puzzles/rush" },
        ]}
      />
      <NavbarButton
        title="Community"
        link="community"
        options={[
          { title: "Friends", link: "community" },
          { title: "Clubs", link: "clubs" },
          { title: "Members", link: "members" },
        ]}
      />
    </nav>
  );
}

export default Navbar;
