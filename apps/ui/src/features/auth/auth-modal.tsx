import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useAuth } from "../../stores/AuthContext";
import { useAuthModal } from "../../stores/AuthModalContext";
import { loginFormSchema, type LoginFormData } from "../../data/authSchema";
import { Link } from "react-router";
import { Modal } from "../../components/ui/modal";
import { TextField } from "../../components/ui/text-field";
import { Button } from "../../components/ui/button";

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
    <Modal labelledBy="auth-modal-title" onClose={closeModal}>
      <div className="bg-options-header px-6 py-5">
        <h2 id="auth-modal-title" className="text-xl font-extrabold">
          Log in to continue
        </h2>
        <p className="mt-1 text-sm text-clock-disabled">You need an account to do that.</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-4 p-6">
        <TextField id="modal-email" type="email" label="Email" placeholder="Email" autoComplete="email" error={errors.email?.message} {...register("email")} />
        <TextField
          id="modal-password"
          type="password"
          label="Password"
          placeholder="Password"
          autoComplete="current-password"
          error={errors.password?.message}
          {...register("password")}
        />

        {errors.root && (
          <p role="alert" className="text-left text-sm text-danger">
            {errors.root.message}
          </p>
        )}

        <Button type="submit" size="lg" disabled={isSubmitting} className="w-full">
          {isSubmitting ? "Logging in..." : "Log in"}
        </Button>
        <div className="flex items-center justify-between text-sm">
          <Link to="/register" onClick={closeModal} className="font-extrabold text-accent hover:text-accent-hover">
            Sign up
          </Link>
          <Button variant="ghost" size="sm" onClick={closeModal}>
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
}
