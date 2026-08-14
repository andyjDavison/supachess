import NavbarButton from "./navbar-button";

function Navbar() {
  return (
    <nav className="flex flex-col fixed top-0 left-0 w-1/8 h-screen bg-nav-bg">
      <h2 className="flex items-center pl-7 h-14">Supachess</h2>
      <NavbarButton
        title="Play"
        link="/play"
        options={[
          { title: "Play Online", link: "play/online" },
          { title: "Play Bots", link: "play/computer" },
        ]}
      />
      <NavbarButton
        title="Play"
        link="/play"
        options={[
          { title: "Play Online", link: "play/online" },
          { title: "Play Bots", link: "play/computer" },
        ]}
      />
      <NavbarButton
        title="Play"
        link="/play"
        options={[
          { title: "Play Online", link: "play/online" },
          { title: "Play Bots", link: "play/computer" },
        ]}
      />
      <NavbarButton
        title="Play"
        link="/play"
        options={[
          { title: "Play Online", link: "play/online" },
          { title: "Play Bots", link: "play/computer" },
        ]}
      />
    </nav>
  );
}

export default Navbar;
