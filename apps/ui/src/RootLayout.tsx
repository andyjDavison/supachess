import { Outlet } from "react-router";
import Navbar from "./components/navbar/navbar";

function RootLayout() {
  return (
    <div className="flex items-center justify-center h-screen w-screen">
      <Navbar />
      <div className="h-full flex-1 z-10 p-4">
        <Outlet />
      </div>
    </div>
  );
}

export default RootLayout;
