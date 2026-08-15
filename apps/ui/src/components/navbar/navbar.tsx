import { Link } from "react-router";
import NavbarButton from "./navbar-button";

function Navbar() {
  return (
    <nav className="flex flex-col fixed top-0 left-0 w-38 h-screen bg-nav-bg z-50 justify-between">
      <div>
        <h2 className="flex items-center pl-7 h-14">Supachess</h2>
        <NavbarButton
          title="Play"
          link="play"
          options={[
            { title: "Play Online", link: "/play/online" },
            { title: "Play Bots", link: "/play/computer" },
          ]}
        />
        <NavbarButton
          title="Puzzles"
          link="puzzles"
          options={[
            { title: "Puzzles", link: "/puzzles" },
            { title: "Daily Puzzles", link: "/daily" },
            { title: "Puzzle Rush", link: "/rush" },
          ]}
        />
        <NavbarButton
          title="Community"
          link="community"
          options={[
            { title: "Friends", link: "/community" },
            { title: "Clubs", link: "/clubs" },
            { title: "Members", link: "/members" },
          ]}
        />
      </div>
      <div className="flex flex-col w-full items-center gap-3 pb-3">
        <Link
          to="/register"
          className="flex items-center justify-center w-9/10 h-9 rounded-sm bg-linear-to-b from-fuchsia-400 to-fuchsia-500 hover:bg-linear-to-b hover:from-fuchsia-300 hover:to-fuchsia-500"
        >
          <p className="text-white/90 font-extrabold text-xs">Sign Up</p>
        </Link>
        <Link
          to="/login"
          className="flex items-center justify-center w-9/10 h-9 rounded-sm bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end"
        >
          <p className="text-white/90 font-extrabold text-xs">Log In</p>
        </Link>
      </div>
    </nav>
  );
}

export default Navbar;
