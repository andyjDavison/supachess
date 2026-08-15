import { Outlet } from "react-router";

function PlayLayout() {
  return (
    <div className="flex items-center justify-center h-screen">
      <Outlet />
    </div>
  );
}

export default PlayLayout;
