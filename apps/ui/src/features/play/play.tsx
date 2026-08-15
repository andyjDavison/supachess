import { Chessboard, type ChessboardOptions } from "react-chessboard";
import { Link } from "react-router";

type PlayOption = {
  title: string;
  desc: string;
  link: string;
};

const playOptions: PlayOption[] = [
  {
    title: "Play Online",
    desc: "Play vs a person of similar skill",
    link: "online",
  },
  {
    title: "Play Bots",
    desc: "Challenge a bot from Easy to Master",
    link: "computer",
  },
  {
    title: "Play Coach",
    desc: "Learn as you play a game with Coach",
    link: "",
  },
  {
    title: "Play a Friend",
    desc: "Invite a friend to a game of chess",
    link: "",
  },
  {
    title: "Tournaments",
    desc: "Join an Arena where anyone can win",
    link: "",
  },
  {
    title: "Chess Variants",
    desc: "Find fun new ways to play chess",
    link: "",
  },
];

function Play() {
  const chessboardOptions: ChessboardOptions = {
    allowDragging: false,
    showAnimations: false,
  };

  return (
    <div className="flex items-center justify-center w-full h-screen gap-7">
      <div className="w-18/40 hover:cursor-default">
        <Chessboard options={chessboardOptions} />
      </div>
      <div className="h-full w-110 py-3">
        <div className="p-4 bg-options-header rounded-t-sm">
          <h1 className="text-3xl font-extrabold text-white">Play Chess</h1>
        </div>
        <div className="flex flex-col gap-2 pt-5 px-5 bg-nav-bg rounded-b-sm pb-26">
          {playOptions.map((option) => (
            <Link
              to={option.link}
              className="flex flex-col text-start pl-17 py-5 rounded-lg gap-.5 bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end"
            >
              <h2 className="text-xl font-extrabold text-white">
                {option.title}
              </h2>
              <p className="text-sm font-medium text-white/70">{option.desc}</p>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}

export default Play;
