import { useNavigate } from "react-router";
import { useGameStore } from "./store";
import { useAuth } from "../../stores/AuthContext";
import type { Game, SideRating } from "../../lib/ws/schemas";
import { describeOutcome } from "./describe-outcome";
import { usePlayers, type Player } from "./use-players";
import { presetForTimeControl } from "../../lib/presets";
import { Modal } from "../../components/ui/modal";
import { Button } from "../../components/ui/button";
import white400 from "../../assets/white_400.png";
import black400 from "../../assets/black_400.png";

interface GameOverModalProps {
  game: Game;
}

export function GameOverModal({ game }: GameOverModalProps) {
  const navigate = useNavigate();
  const { user } = useAuth();
  const clearMatch = useGameStore((s) => s.clearMatch);
  const joinQueue = useGameStore((s) => s.joinQueue);
  const { white, black } = usePlayers(game.whiteId, game.blackId);

  const isWhite = user?.id === game.whiteId;
  const { headline, detail } = describeOutcome(game, isWhite);
  const preset = presetForTimeControl(game.initialMs, game.incrementMs);

  function handleBackToLobby() {
    clearMatch(); // defensive - clears any stale matchmaking state before leaving
    navigate("/play/online");
  }

  // Straight back into the queue at the same time control.
  function handleNewGame() {
    clearMatch();
    if (preset) joinQueue(preset);
    navigate("/play/online");
  }

  return (
    <Modal labelledBy="game-over-title">
      <div className="bg-options-header px-6 py-5 text-center">
        <h2 id="game-over-title" className="text-2xl font-extrabold">
          {headline}
        </h2>
        {detail && <p className="mt-1 text-sm text-clock-disabled">{detail}</p>}
      </div>

      <div className="flex items-start justify-center gap-6 px-6 pt-5">
        <PlayerResult avatar={white400} player={white} rating={game.ratings?.white} winner={game.result === "white_wins"} />
        <span className="pt-5 text-sm text-clock-disabled">vs</span>
        <PlayerResult avatar={black400} player={black} rating={game.ratings?.black} winner={game.result === "black_wins"} />
      </div>

      <div className="flex flex-col gap-3 p-6">
        {preset && (
          <Button size="lg" onClick={handleNewGame}>
            New game
          </Button>
        )}
        <Button variant="secondary" onClick={handleBackToLobby}>
          Back to lobby
        </Button>
        {/* Placeholder until Post Game Analysis exists. */}
        <Button variant="ghost" size="sm" disabled title="Coming soon">
          Analyze game
        </Button>
      </div>
    </Modal>
  );
}

function PlayerResult({
  avatar,
  player,
  rating,
  winner,
}: {
  avatar: string;
  player: Player | null;
  rating: SideRating | undefined;
  winner: boolean;
}) {
  const delta = rating?.delta;
  return (
    <div className="flex w-24 flex-col items-center gap-1">
      <img src={avatar} alt="" className={`size-14 rounded-md ${winner ? "ring-2 ring-success" : ""}`} />
      <p className="max-w-full truncate text-sm">{player?.username ?? "…"}</p>
      {rating && (
        <p className="text-xs text-clock-disabled">
          {rating.rating + (delta ?? 0)}
          {delta !== undefined && (
            <span className={delta >= 0 ? "ml-1 text-success" : "ml-1 text-danger"}>
              {delta >= 0 ? `+${delta}` : delta}
            </span>
          )}
        </p>
      )}
    </div>
  );
}
