import { Outlet } from "react-router";
import Navbar from "../../components/navbar/navbar";

function PlayLayout() {
  return (
    <div className="flex items-center justify-center h-screen">
      <Navbar />
      <div className="h-full w-10/11 z-10">
        <Outlet />
      </div>
    </div>
  );
}

export default PlayLayout;
