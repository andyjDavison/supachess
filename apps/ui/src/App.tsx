import { Outlet } from "react-router";
import "./index.css";

function App() {
  return (
    <main className="h-full w-full overflow-hidden flex items-center justify-center">
      <Outlet />
    </main>
  );
}

export default App;
