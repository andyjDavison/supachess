import { createBrowserRouter } from "react-router";
import App from "./App";
import Play from "./features/play/play";
import PlayComputer from "./features/play/play-computer";
import PlayOnline from "./features/play/play-online";
import PlayLayout from "./features/play/play-layout";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: App,
    children: [
      {
        path: "play",
        Component: PlayLayout,
        children: [
          {
            index: true,
            Component: Play,
          },
          { path: "online", Component: PlayOnline },
          { path: "computer", Component: PlayComputer },
        ],
      },
    ],
  },
]);
