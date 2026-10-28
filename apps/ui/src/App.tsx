import { Outlet } from "react-router";
import "./index.css";
import { useAuth } from "./stores/AuthContext";
import { InviteToasts } from "./features/invites/invite-toasts";

function App() {
  const { user } = useAuth();

  console.log(user);

  return (
    <main className="h-full w-full overflow-hidden flex items-center justify-center">
      <Outlet />
      {user && <InviteToasts />}
    </main>
  );
}

export default App;
