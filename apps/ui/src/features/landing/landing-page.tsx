import { Link } from "react-router";
import { Chessboard } from "react-chessboard";
import { useAuth } from "../../stores/AuthContext";
import { useBoardAppearance } from "../../stores/board-settings";
import { buttonClass } from "../../components/ui/button-styles";
import puzzlePiece from "../../assets/puzzle-piece.svg";
import robot from "../../assets/device-bot.svg";
import friends from "../../assets/friends.svg";

// A middlegame (an Italian Game), so the hero board isn't the empty start.
const HERO_FEN = "r1bq1rk1/pppp1ppp/2n2n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7";

const TILES = [
  { title: "Solve Puzzles", desc: "Sharpen your tactics one position at a time.", link: "/puzzles", img: puzzlePiece },
  { title: "Play Bots", desc: "Practice against the computer at your own pace.", link: "/play/computer", img: robot },
  { title: "Join the Community", desc: "Find friends, clubs and other players.", link: "/community", img: friends },
];

export default function LandingPage() {
  const { user } = useAuth();
  const appearance = useBoardAppearance();

  return (
    <div className="mx-auto flex h-full w-full max-w-6xl flex-col gap-12 overflow-y-auto px-8 py-10">
      <section className="flex items-center gap-16">
        <div className="w-1/2 max-w-lg shrink-0">
          <Chessboard
            options={{
              id: "landing",
              position: HERO_FEN,
              allowDragging: false,
              showAnimations: false,
              boardStyle: { borderRadius: 6, pointerEvents: "none", boxShadow: "0 12px 40px rgba(0,0,0,0.45)" },
              ...appearance,
              showNotation: false,
            }}
          />
        </div>
        <div className="flex flex-1 flex-col items-center gap-6 text-center">
          <h1 className="text-5xl leading-tight font-extrabold text-white">Play Chess Online on the #2 Site!</h1>
          <p className="text-lg text-clock-disabled">Join 0 players in the world's smallest chess community</p>
          <div className="flex w-full max-w-sm flex-col gap-3">
            {user ? (
              <Link to="/play/online" className={buttonClass("primary", "lg", "w-full")}>
                Play Online
              </Link>
            ) : (
              <Link to="/register" className={buttonClass("primary", "lg", "w-full")}>
                Get Started
              </Link>
            )}
            <Link to="/play/computer" className={buttonClass("secondary", "lg", "w-full")}>
              Play Bots
            </Link>
          </div>
          {!user && (
            <p className="text-sm text-clock-disabled">
              Already have an account?{" "}
              <Link to="/login" className="font-extrabold text-accent hover:text-accent-hover">
                Log In
              </Link>
            </p>
          )}
        </div>
      </section>

      <section aria-label="More to do" className="grid grid-cols-3 gap-4">
        {TILES.map((tile) => (
          <Link
            key={tile.title}
            to={tile.link}
            className="flex flex-col items-start gap-3 rounded-xl bg-nav-bg p-6 text-left transition-colors hover:bg-options-button"
          >
            <img src={tile.img} alt="" className="size-10" />
            <h2 className="text-xl font-extrabold text-white">{tile.title}</h2>
            <p className="text-sm text-clock-disabled">{tile.desc}</p>
          </Link>
        ))}
      </section>
    </div>
  );
}
