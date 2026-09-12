import { useMemo } from "react";
import { Chess, Move, type Square } from "chess.js";
import {
  Chessboard,
  type ChessboardOptions,
  type PieceDropHandlerArgs,
} from "react-chessboard";
import { useGameStore } from "./store";
import type { Game } from "../../lib/ws/schemas";
import { useAuth } from "../../stores/AuthContext";

interface GameBoardProps {
  game: Game;
}

export function GameBoard({ game }: GameBoardProps) {
  const submitMove = useGameStore((s) => s.submitMove);
  const resign = useGameStore((s) => s.resign);
  const { user } = useAuth();

  // A fresh chess.js instance per FEN, purely for local legality
  // checking on drag - this never becomes the source of truth for
  // what's actually rendered. The board's real position comes from
  // game.fen (server-confirmed), not from this instance.
  const localChess = useMemo(() => new Chess(game.fen), [game.fen]);

  function onPieceDrop({
    sourceSquare,
    targetSquare,
  }: PieceDropHandlerArgs): boolean {
    if (!targetSquare) return false;

    const legalMoves = localChess.moves({
      square: sourceSquare as Square,
      verbose: true,
    });
    const isLegal = legalMoves.some((m) => m.to === targetSquare);
    if (!isLegal) return false;

    const isPromotion = legalMoves.some(
      (m) => m.to === targetSquare && m.flags.includes("p"),
    );

    submitMove(sourceSquare, targetSquare, isPromotion ? "q" : undefined);

    return false;
  }

  const chessboardOptions: ChessboardOptions = {
    position: game.fen,
    onPieceDrop,
    boardOrientation: user?.id === game.whiteId ? "white" : "black",
  };

  return (
    <div className="flex flex-col items-center gap-4 p-8">
      <Chessboard options={chessboardOptions} />
      <button
        onClick={resign}
        className="rounded-md border border-red-300 px-4 py-2 text-sm text-red-600"
      >
        Resign
      </button>
    </div>
  );
}
