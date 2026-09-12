import { createBrowserRouter } from "react-router";
import App from "../App";
import Play from "../features/play/play";
import PlayComputer from "../features/play/play-computer";
import PlayOnline from "../features/play/play-online";
import Signup from "../features/auth/signup";
import Community from "../features/community/community";
import Clubs from "../features/community/clubs";
import Friends from "../features/community/friends";
import Members from "../features/community/members";
import Puzzles from "../features/puzzle/puzzles";
import PuzzleRush from "../features/puzzle/puzzle-rush";
import DailyPuzzle from "../features/puzzle/daily-puzzles";
import RootLayout from "../RootLayout";
import LandingPage from "../features/landing/landing-page";
import Login from "../features/auth/login";
import { ProtectedLayout } from "./ProtectedRoute";
import GamePage from "../features/game/game";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      {
        path: "/",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: LandingPage,
          },
        ],
      },
      {
        path: "register",
        Component: Signup,
      },
      {
        path: "login",
        Component: Login,
      },
      {
        path: "play",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: Play,
          },
          { path: "online", Component: PlayOnline },
          { path: "computer", Component: PlayComputer },
        ],
      },
      {
        Component: ProtectedLayout,
        children: [
          {
            path: "game/:gameId",
            Component: RootLayout,
            children: [{ index: true, Component: GamePage }],
          },
        ],
      },
      {
        path: "puzzles",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: Puzzles,
          },
          {
            path: "rush",
            Component: PuzzleRush,
          },
        ],
      },
      {
        path: "daily",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: DailyPuzzle,
          },
        ],
      },
      {
        path: "community",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: Community,
          },
        ],
      },
      {
        path: "friends",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: Friends,
          },
        ],
      },
      {
        path: "clubs",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: Clubs,
          },
        ],
      },
      {
        path: "members",
        Component: RootLayout,
        children: [
          {
            index: true,
            Component: Members,
          },
        ],
      },
    ],
  },
]);
