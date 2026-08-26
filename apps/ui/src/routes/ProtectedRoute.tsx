import { Navigate, Outlet } from "react-router";
import { useAuth } from "../stores/AuthContext.tsx";

/**
 * A layout route, not a wrapper around a single page. Mount this as the
 * `element` of a parent route with no `path`, and put every route that
 * needs auth as its `children` - one guard protects all of them via
 * <Outlet />, instead of wrapping each page's element individually.
 */
export function ProtectedLayout() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
