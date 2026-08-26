import { useAuth } from "../../stores/AuthContext";
import { useAuthModal } from "../../stores/AuthModalContext";

/**
 * Returns a `requireAuth` function you wrap around any action that needs a
 * logged-in user - a button's onClick, typically. If the user is logged in,
 * the action runs immediately. If not, the login modal opens with this
 * action queued up to run automatically the moment login succeeds.
 *
 * Unlike a redirect-based guard, this never navigates the user away from
 * the page they were on - the page (and whatever they were about to do)
 * stays right where it was.
 */
export function useRequireAuth() {
  const { user, isLoading } = useAuth();
  const { openModal } = useAuthModal();

  return (action: () => void) => {
    // Ignore the click rather than guess - clicking before the initial
    // /me check resolves would otherwise look identical to "logged out"
    // and pop the modal for an already-logged-in user.
    if (isLoading) return;

    if (!user) {
      openModal(action);
      return;
    }

    action();
  };
}
