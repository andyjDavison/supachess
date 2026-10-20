import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useAuth } from "../../stores/AuthContext";
import { useAuthModal } from "../../stores/AuthModalContext";
import { loginFormSchema, type LoginFormData } from "../../data/authSchema";

export function AuthModal() {
  const { isOpen, closeModal, pendingAction } = useAuthModal();
  const { login } = useAuth();

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({ resolver: zodResolver(loginFormSchema) });

  useEffect(() => {
    if (isOpen) reset();
  }, [isOpen, reset]);

  useEffect(() => {
    if (!isOpen) return;
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") closeModal();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, closeModal]);

  if (!isOpen) return null;

  const onSubmit = async (values: LoginFormData) => {
    try {
      await login(values);
    } catch {
      setError("root", { message: "Invalid email or password" });
      return;
    }
    pendingAction?.();
    closeModal();
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={closeModal}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="auth-modal-title"
        className="w-full max-w-sm rounded-lg border border-input-border bg-nav-bg p-6 text-text shadow-xl"
        onClick={(event) => event.stopPropagation()}
      >
        <h2 id="auth-modal-title" className="text-lg font-semibold">
          Log in to continue
        </h2>
        <p className="mt-1 text-sm text-clock-black-d">
          You need an account to do that.
        </p>

        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          className="mt-4 space-y-4"
        >
          <div>
            <label htmlFor="modal-email" className="block text-sm font-medium">
              Email
            </label>
            <input
              id="modal-email"
              type="email"
              {...register("email")}
              aria-invalid={!!errors.email}
              className="mt-1 w-full rounded-md border border-input-border bg-input-bg px-3 py-2 text-text placeholder:text-clock-black-d focus:outline-none focus:ring-2 focus:ring-accent aria-invalid:border-red-500 aria-invalid:focus:ring-red-500"
            />
            {errors.email && (
              <p role="alert" className="mt-1 text-sm text-red-400">
                {errors.email.message}
              </p>
            )}
          </div>

          <div>
            <label
              htmlFor="modal-password"
              className="block text-sm font-medium"
            >
              Password
            </label>
            <input
              id="modal-password"
              type="password"
              {...register("password")}
              aria-invalid={!!errors.password}
              className="mt-1 w-full rounded-md border border-input-border bg-input-bg px-3 py-2 text-text placeholder:text-clock-black-d focus:outline-none focus:ring-2 focus:ring-accent aria-invalid:border-red-500 aria-invalid:focus:ring-red-500"
            />
            {errors.password && (
              <p role="alert" className="mt-1 text-sm text-red-400">
                {errors.password.message}
              </p>
            )}
          </div>

          {errors.root && (
            <p role="alert" className="text-sm text-red-400">
              {errors.root.message}
            </p>
          )}

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={closeModal}
              className="px-3 py-2 text-sm text-clock-black-d hover:text-text hover:cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="rounded-md bg-linear-to-b from-accent to-accent-strong px-4 py-2 text-sm font-medium text-text hover:cursor-pointer hover:from-accent-hover disabled:opacity-50"
            >
              {isSubmitting ? "Logging in..." : "Log in"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
