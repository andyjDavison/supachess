import { zodResolver } from "@hookform/resolvers/zod";
import { loginFormSchema, type LoginFormData } from "../../data/authSchema";
import { useForm } from "react-hook-form";
import { useAuth } from "../../stores/AuthContext";
import { Link, useNavigate } from "react-router";
import { AuthLayout, ProviderButtons } from "./auth-layout";
import { TextField } from "../../components/ui/text-field";
import { Button } from "../../components/ui/button";

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginFormSchema),
    mode: "onSubmit",
  });

  const onSubmit = async (data: LoginFormData) => {
    try {
      await login(data);
      navigate("/");
    } catch {
      setError("root", {
        message: "Incorrect username or password. Please try again.",
      });
    }
  };

  return (
    <AuthLayout
      footer={
        <>
          New?{" "}
          <Link to="/register" className="font-extrabold text-accent hover:text-accent-hover">
            Sign up and start playing chess!
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-4 p-8">
        <h1 className="text-2xl font-extrabold">Log In</h1>
        <TextField id="email" type="email" label="Email" placeholder="Email" autoComplete="email" error={errors.email?.message} {...register("email")} />
        <TextField
          id="password"
          type="password"
          label="Password"
          placeholder="Password"
          autoComplete="current-password"
          error={errors.password?.message}
          {...register("password")}
        />
        {/* onSubmit sets a root error on a failed login. */}
        {errors.root && (
          <p role="alert" className="text-left text-sm text-danger">
            {errors.root.message}
          </p>
        )}
        <Button type="submit" size="lg" disabled={isSubmitting} className="mt-2 w-full">
          Log In
        </Button>
        <ProviderButtons providers={["Google", "Apple"]} />
      </form>
    </AuthLayout>
  );
}
