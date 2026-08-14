import { Outlet } from "react-router";
import "./index.css";
import Navbar from "./components/navbar/navbar";

function App() {
  return (
    <main className="h-screen w-screen overflow-hidden flex items-center justify-center">
      <Navbar />
      <div className="w-1/3">
        <Outlet />
      </div>
    </main>
  );
}

export default App;
