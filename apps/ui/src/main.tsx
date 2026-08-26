import { createRoot } from "react-dom/client";
import "./index.css";
import { RouterProvider } from "react-router";
import { router } from "./routes/routes.ts";
import { AuthProvider } from "./stores/AuthContext.tsx";
import { AuthModalProvider } from "./stores/AuthModalContext.tsx";
import { AuthModal } from "./features/auth/auth-modal.tsx";

createRoot(document.getElementById("root")!).render(
  <AuthProvider>
    <AuthModalProvider>
      <RouterProvider router={router} />
      <AuthModal />
    </AuthModalProvider>
  </AuthProvider>,
);
