import { Chessboard, type ChessboardOptions } from "react-chessboard";
import { Link } from "react-router";
import black400 from "../../assets/black_400.png";
import white400 from "../../assets/white_400.png";
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
    link: "",
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
  const chessboardOptions: ChessboardOptions = {
    allowDragging: false,
    showAnimations: false,
  };

  return (
    <div className="flex items-center justify-center w-full h-full gap-7">
      <div className="flex flex-col w-23/40 hover:cursor-default gap-2">
        <div className="flex gap-2 w-full h-9">
          <img src={black400} className="rounded-sm" />
          <p className="text-white text-xs font-extrabold">Opponent</p>
        </div>
        <Chessboard options={chessboardOptions} />
        <div className="flex gap-2 w-full h-9">
          <img src={white400} className="rounded-sm" />
          <p className="text-white text-xs font-extrabold">Player</p>
        </div>
      </div>
      <div className="flex flex-col h-full flex-1">
        <div className="bg-options-header rounded-t-sm h-15">
          <h1 className="flex items-center justify-center h-full text-3xl font-extrabold text-white">
            Play Chess
          </h1>
        </div>
        <div className="flex flex-col flex-1 gap-2 py-5 px-5 bg-nav-bg rounded-b-sm">
          {playOptions.map((option) => (
            <Link
              to={option.link}
              className="flex flex-row items-center gap-4 pl-5 py-5 rounded-lg gap-.5 bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end"
            >
              <img src={option.img} className="size-11" />
              <div className="flex flex-col text-start">
                <h2 className="text-xl font-extrabold text-white">
                  {option.title}
                </h2>
                <p className="text-sm font-medium text-white/70">
                  {option.desc}
                </p>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}

export default Play;
