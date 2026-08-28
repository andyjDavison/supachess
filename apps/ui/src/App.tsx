import { Outlet } from "react-router";
import "./index.css";
import { useAuth } from "./stores/AuthContext";

function App() {
  const { user } = useAuth();

  console.log(user);

  return (
    <main className="h-full w-full overflow-hidden flex items-center justify-center">
      <Outlet />
    </main>
  );
}

export default App;
