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
import { PlayerBar } from "./player-bar";
import { BoardSettingsButton } from "./board-settings";
import { usePlayers } from "./use-players";
import { material } from "../../lib/chess/material";
import { useBoardAppearance } from "../../stores/board-settings";

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
  const appearance = useBoardAppearance();
  const players = usePlayers(game.whiteId, game.blackId);
  const { capturedByWhite, capturedByBlack, balance } = useMemo(() => material(game.fen), [game.fen]);

  // The game's own snapshot is the rating for its speed; profiles only
  // carry the Blitz one, so they're the fallback for older games.
  const sides = {
    white: {
      name: players.white?.username ?? (isWhitePlayer ? user?.username : opponent?.username) ?? "White",
      rating: game.ratings?.white.rating ?? players.white?.rating,
      captured: capturedByWhite,
      advantage: balance,
      clock: { ms: whiteMs, isActive: activeColor === "w" },
    },
    black: {
      name: players.black?.username ?? (isWhitePlayer ? opponent?.username : user?.username) ?? "Black",
      rating: game.ratings?.black.rating ?? players.black?.rating,
      captured: capturedByBlack,
      advantage: -balance,
      clock: { ms: blackMs, isActive: activeColor === "b" },
    },
  };
  const [topColor, bottomColor] = isWhitePlayer ? (["black", "white"] as const) : (["white", "black"] as const);

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
    ...appearance,
  };

  return (
    // --board-chrome is the two h-11 player bars (5.5rem) plus the two
    // gap-2 gaps (1rem). .board-square subtracts it from the container
    // height so the board shrinks on either axis.
    <div className="flex flex-col items-center gap-2 w-full [--board-chrome:6.5rem]">
      <div className="board-square mx-auto flex flex-col gap-2 aspect-auto!">
        <PlayerBar color={topColor} {...sides[topColor]} />
        <div className="relative">
          <Chessboard options={chessboardOptions} />
          <div className="absolute top-0 -right-11">
            <BoardSettingsButton />
          </div>
        </div>
        <PlayerBar color={bottomColor} {...sides[bottomColor]} />
      </div>
    </div>
  );
}
