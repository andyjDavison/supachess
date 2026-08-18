import { createBrowserRouter } from "react-router";
import App from "./App";
import Play from "./features/play/play";
import PlayComputer from "./features/play/play-computer";
import PlayOnline from "./features/play/play-online";
import Signup from "./features/auth/signup";
import Community from "./features/community/community";
import Clubs from "./features/community/clubs";
import Friends from "./features/community/friends";
import Members from "./features/community/members";
import Puzzles from "./features/puzzle/puzzles";
import PuzzleRush from "./features/puzzle/puzzle-rush";
import DailyPuzzle from "./features/puzzle/daily-puzzles";
import RootLayout from "./RootLayout";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      {
        index: true,
        Component: RootLayout,
      },
      {
        path: "register",
        Component: Signup,
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
