import { Link } from "react-router";
import vid from "../../assets/landing_vid.webm";

export default function LandingPage() {
  return (
    <div className="flex flex-row w-full px-20">
      <div className="w-1/2">
        <video src={vid} className="rounded-lg" />
      </div>
      <div className="flex flex-col gap-4 w-2/5 pl-20 items-center justify-center">
        <h1 className="text-4xl text-white">
          Play Chess Online on the #2 Site!
        </h1>
        <p className="text-white/90">
          Join 0 players in the world's smallest chess community
        </p>
        <Link
          to={"/register"}
          className="flex items-center justify-center w-9/10 h-15 rounded-lg bg-linear-to-b from-accent to-accent-strong hover:bg-linear-to-b hover:from-accent-hover hover:to-accent-strong"
        >
          <p className="text-white/90 font-extrabold text-xl">Get Started</p>
        </Link>
      </div>
    </div>
  );
}
