import { Outlet } from "react-router";
import "./index.css";
import { useAuth } from "./stores/AuthContext";
import { InviteToasts } from "./features/invites/invite-toasts";
import { MessagesBootstrap } from "./features/messages/messages-bootstrap";

function App() {
  const { user } = useAuth();

  console.log(user);

  return (
    <main className="h-full w-full overflow-hidden flex items-center justify-center">
      <Outlet />
      {user && <InviteToasts />}
      {user && <MessagesBootstrap />}
    </main>
  );
}

export default App;
