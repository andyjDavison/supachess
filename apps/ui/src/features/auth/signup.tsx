import { Link } from "react-router";

function Signup() {
  return (
    <div className="flex flex-col gap-4 pt-4 items-center">
      <header>
        <Link to="/" className="text-white/90 text-2xl font-extrabold">
          Supachess.com!
        </Link>
      </header>
      <div className="flex flex-col items-center justify-between h-140 w-40/50">
        <h1 className="text-3xl text-white/90 font-extrabold">
          Create Your Supachess.com Account
        </h1>
        <div className="flex flex-col w-full items-center gap-2">
          <button className="w-4/5 p-4 text-white/90 text-xl font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-fuchsia-400 to-fuchsia-500 hover:bg-linear-to-b hover:from-fuchsia-300 hover:to-fuchsia-500">
            Contine with Email
          </button>
          <p className="text-xs m-3">OR</p>
          <button className="w-4/5 p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end">
            Contine with Phone
          </button>
          <button className="w-4/5 p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end">
            Contine with Google
          </button>
          <button className="w-4/5 p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end">
            Contine with Apple
          </button>
        </div>
      </div>
    </div>
  );
}

export default Signup;
