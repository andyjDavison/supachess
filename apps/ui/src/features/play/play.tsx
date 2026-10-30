import { Link } from "react-router";
import { PreviewBoard } from "./preview-board";
import { useAuth } from "../../stores/AuthContext";
import chessPiece from "../../assets/play-white.svg";
import lightning from "../../assets/time-blitz.svg";
import robot from "../../assets/device-bot.svg";
import coach from "../../assets/coachdavid-icon.png";
import handShake from "../../assets/hand-shake.svg";
import tournament from "../../assets/tournaments.svg";
import variant from "../../assets/variants.svg";

type PlayOption = {
  title: string;
  desc: string;
  link: string;
  img: string;
};

const playOptions: PlayOption[] = [
  {
    title: "Play Online",
    desc: "Play vs a person of similar skill",
    link: "online",
    img: lightning,
  },
  {
    title: "Play Bots",
    desc: "Challenge a bot from Easy to Master",
    link: "computer",
    img: robot,
  },
  {
    title: "Play Coach",
    desc: "Learn as you play a game with Coach",
    link: "",
    img: coach,
  },
  {
    title: "Play a Friend",
    desc: "Invite a friend to a game of chess",
    link: "/friends",
    img: handShake,
  },
  {
    title: "Tournaments",
    desc: "Join an Arena where anyone can win",
    link: "",
    img: tournament,
  },
  {
    title: "Chess Variants",
    desc: "Find fun new ways to play chess",
    link: "",
    img: variant,
  },
];

function Play() {
  const { user } = useAuth();
  return (
    <div className="game-layout">
      <div className="board-column">
        <PreviewBoard opponent="Opponent" me={user?.username ?? "Player"} />
      </div>
      <div className="side-panel">
        <div className="flex h-16 shrink-0 items-center justify-center gap-2 rounded-t-md bg-options-header">
          <img src={chessPiece} alt="" className="size-8" />
          <h1 className="text-2xl font-extrabold text-white">Play Chess</h1>
        </div>
        <div className="flex flex-1 flex-col gap-2 overflow-y-auto rounded-b-md bg-nav-bg p-4">
          {playOptions.map((option) =>
            option.link ? (
              <Link
                key={option.title}
                to={option.link}
                className="flex items-center gap-4 rounded-lg bg-linear-to-b from-gradient-start to-gradient-end p-5 shadow-[0_4px_0_0_var(--color-button-shadow)] hover:from-hover-gradient-start hover:to-hover-gradient-end"
              >
                <PlayOptionBody option={option} />
              </Link>
            ) : (
              // Not built yet: shown so the menu reads like the real thing.
              <div
                key={option.title}
                aria-disabled="true"
                title="Coming soon"
                className="flex items-center gap-4 rounded-lg bg-options-button/50 p-5 opacity-50"
              >
                <PlayOptionBody option={option} />
              </div>
            ),
          )}
        </div>
      </div>
    </div>
  );
}

function PlayOptionBody({ option }: { option: PlayOption }) {
  return (
    <>
      <img src={option.img} alt="" className="size-11" />
      <div className="flex flex-col text-start">
        <h2 className="text-xl font-extrabold text-white">{option.title}</h2>
        <p className="text-sm font-medium text-white/70">{option.desc}</p>
      </div>
    </>
  );
}

export default Play;
