import { Outlet } from "react-router";
import "./index.css";
import Navbar from "./components/navbar/navbar";

function App() {
  return (
    <main className="h-full w-full overflow-hidden flex items-center justify-end">
      <Navbar />
      <div className="h-full w-10/11 z-10">
        <Outlet />
      </div>
    </main>
  );
}

export default App;
