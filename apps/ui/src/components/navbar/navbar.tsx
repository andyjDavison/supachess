import { Link } from "react-router";
import NavbarButton from "./navbar-button";
import puzzlePiece from "../../assets/puzzle-piece.svg";
import chessPiece from "../../assets/play-white.svg";
import friends from "../../assets/friends.svg";
import robot from "../../assets/device-bot.svg";
import calendar from "../../assets/calendar-dailypuzzle.svg";
import rush from "../../assets/puzzle-rush.svg";
import clubs from "../../assets/clubs.svg";
import globe from "../../assets/globe.svg";
import { useAuth } from "../../stores/AuthContext";

const playSubmenu = [
  { title: "Play Online", link: "/play/online", img: chessPiece },
  { title: "Play Bots", link: "/play/computer", img: robot },
];

const puzzlesSubmenu = [
  { title: "Puzzles", link: "/puzzles", img: puzzlePiece },
  { title: "Daily Puzzles", link: "/daily", img: calendar },
  { title: "Puzzle Rush", link: "/puzzles/rush", img: rush },
];

const communitySubmenu = [
  { title: "Friends", link: "/friends", img: friends },
  { title: "Clubs", link: "/clubs", img: clubs },
  { title: "Members", link: "/members", img: globe },
];

function Navbar() {
  const auth = useAuth();

  return (
    <nav className="flex flex-col shrink-0 w-38 h-screen bg-nav-bg z-50 justify-between">
      <div>
        <h2 className="flex items-center pl-7 h-14 font-extrabold">
          Supachess
        </h2>
        <NavbarButton
          title="Play"
          link="/play"
          options={playSubmenu}
          img={chessPiece}
        />
        <NavbarButton
          title="Puzzles"
          link="/puzzles"
          options={puzzlesSubmenu}
          img={puzzlePiece}
        />
        <NavbarButton
          title="Community"
          link="/community"
          options={communitySubmenu}
          img={friends}
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
        {auth.user && <button onClick={auth.logout}>logout</button>}
      </div>
    </nav>
  );
}

export default Navbar;
