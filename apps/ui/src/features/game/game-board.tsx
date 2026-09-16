import { useMemo, useState } from "react";
import { Chess, type Square } from "chess.js";
import {
  Chessboard,
  type ChessboardOptions,
  type PieceDropHandlerArgs,
  type PieceHandlerArgs,
  type SquareHandlerArgs,
} from "react-chessboard";
import { useGameStore } from "./store";
import type { Game } from "../../lib/ws/schemas";
import { useAuth } from "../../stores/AuthContext";
import { useLiveClocks } from "./useLiveClocks";
import { ClockDisplay } from "./clock-display";
import black400 from "../../assets/black_400.png";
import white400 from "../../assets/white_400.png";

interface GameBoardProps {
  game: Game;
}

const HIGHLIGHT_COLOR = "rgba(255, 255, 0, 0.4)";
const MARKED_COLOR = "rgba(255, 97, 80, 0.8)";

export function GameBoard({ game }: GameBoardProps) {
  const submitMove = useGameStore((s) => s.submitMove);
  // const resign = useGameStore((s) => s.resign);
  const opponent = useGameStore((s) => s.opponent);
  const { user } = useAuth();
  const { whiteMs, blackMs, activeColor } = useLiveClocks(game);
  const isWhitePlayer = user?.id === game.whiteId;

  const topClock = isWhitePlayer
    ? { ms: blackMs, isActive: activeColor === "b" }
    : { ms: whiteMs, isActive: activeColor === "w" };
  const bottomClock = isWhitePlayer
    ? { ms: whiteMs, isActive: activeColor === "w" }
    : { ms: blackMs, isActive: activeColor === "b" };

  const [selectedSquare, setSelectedSquare] = useState<Square | null>(null);
  const [markedSquares, setMarkedSquares] = useState<Set<string>>(new Set());

  const localChess = useMemo(() => new Chess(game.fen), [game.fen]);

  const legalMoves = useMemo(() => {
    if (!selectedSquare) return [];
    return localChess.moves({
      square: selectedSquare as Square,
      verbose: true,
    });
  }, [localChess, selectedSquare]);

  function tryMove(from: string, to: string): boolean {
    const movesFromSquare = localChess.moves({
      square: from as Square,
      verbose: true as const,
    });
    const matched = movesFromSquare.find((m) => m.to === to);
    if (!matched) return false;

    const isPromotion = matched.flags.includes("p");
    submitMove(from, to, isPromotion ? "q" : undefined);
    setSelectedSquare(null);
    return true;
  }

  function handleSquareInteraction(square: string) {
    const sq = square as Square;

    if (selectedSquare && legalMoves.some((m) => m.to === sq)) {
      tryMove(selectedSquare, sq);
      return;
    }

    const piece = localChess.get(sq);
    const isOwnPiece = piece && piece.color === localChess.turn();
    setSelectedSquare(isOwnPiece ? sq : null);
  }

  function handleSquareClick({ square }: SquareHandlerArgs) {
    handleSquareInteraction(square);
  }

  function handlePieceClick({ square }: PieceHandlerArgs) {
    if (!square) return;
    handleSquareInteraction(square);
  }

  function handlePieceDrop({
    sourceSquare,
    targetSquare,
  }: PieceDropHandlerArgs): boolean {
    if (!targetSquare) {
      setSelectedSquare(null);
      return false;
    }
    tryMove(sourceSquare, targetSquare);
    return false;
  }

  function handleSquareRightClick({ square }: SquareHandlerArgs) {
    setMarkedSquares((prev) => {
      const next = new Set(prev);
      if (next.has(square)) {
        next.delete(square);
      } else {
        next.add(square);
      }
      return next;
    });
  }

  useMemo(() => {
    setMarkedSquares(new Set());
  }, [game.fen]);

  const squareStyles = useMemo(() => {
    const styles: Record<string, React.CSSProperties> = {};

    for (const square of markedSquares) {
      styles[square] = { backgroundColor: MARKED_COLOR };
    }

    if (selectedSquare) {
      styles[selectedSquare] = { backgroundColor: HIGHLIGHT_COLOR };
    }

    for (const move of legalMoves) {
      const isCapture = move.flags.includes("c") || move.flags.includes("e");

      styles[move.to] = isCapture
        ? {
            backgroundColor: HIGHLIGHT_COLOR,
            backgroundImage:
              "radial-gradient(transparent 0%, transparent 79%, rgba(0,0,0,0.3) 80%, rgba(0,0,0,0.3) 88%, transparent 89%)",
          }
        : {
            backgroundImage:
              "radial-gradient(rgba(0,0,0,0.25) 19%, transparent 20%)",
          };
    }

    return styles;
  }, [markedSquares, selectedSquare, legalMoves]);

  const chessboardOptions: ChessboardOptions = {
    position: game.fen,
    onPieceDrop: handlePieceDrop,
    onSquareClick: handleSquareClick,
    onPieceClick: handlePieceClick,
    onSquareRightClick: handleSquareRightClick,
    squareStyles,
    boardOrientation: isWhitePlayer ? "white" : "black",
    boardStyle: {
      borderRadius: 2,
    },
    darkSquareStyle: {
      background: "oklch(74% 0.238 322.16)",
    },
  };

  return (
    <div className="flex flex-col items-center gap-2 p-8">
      <div className="flex gap-2 w-full h-9">
        <img src={black400} className="rounded-xs" />
        <p className="text-white text-xs font-extrabold">
          {opponent?.username}
        </p>
        <div className="flex-1 flex justify-end">
          <ClockDisplay
            isWhite={false}
            ms={topClock.ms}
            isActive={topClock.isActive}
          />
        </div>
      </div>
      <div className="mx-auto w-full max-w-158 aspect-square">
        <Chessboard options={chessboardOptions} />
      </div>
      <div className="flex gap-2 w-full h-9">
        <img src={white400} className="rounded-xs" />
        <p className="text-white text-xs font-extrabold">{user?.username}</p>
        <div className="flex-1 flex w-full justify-end">
          <ClockDisplay
            isWhite={true}
            ms={bottomClock.ms}
            isActive={bottomClock.isActive}
          />
        </div>
      </div>
    </div>
  );
}
