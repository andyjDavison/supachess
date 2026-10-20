import { zodResolver } from "@hookform/resolvers/zod";
import { loginFormSchema, type LoginFormData } from "../../data/authSchema";
import { useForm } from "react-hook-form";
import { useAuth } from "../../stores/AuthContext";
import { useNavigate } from "react-router";

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginFormSchema),
    mode: "onSubmit",
  });

  const onSubmit = async (data: LoginFormData) => {
    try {
      await login(data);
      console.log("logged in");
      navigate("/");
    } catch {
      setError("root", {
        message: "Incorrect username or password. Please try again.",
      });
    }
  };

  return (
    <div>
      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        className="flex flex-col gap-3"
      >
        <div className="flex flex-col items-start">
          <input
            id="email"
            type="text"
            {...register("email")}
            placeholder="Email"
            aria-invalid={!!errors.email}
            aria-describedby={errors.email ? "email-error" : undefined}
            className={`w-full p-1.5 rounded-md text-xs text-text font-semibold border bg-input-bg placeholder:text-clock-black-d ${
              errors.email
                ? "border-red-500 hover:border-red-400"
                : "border-input-border hover:border-gray-500"
            }`}
          />
          {errors.email && (
            <p id="email-error" role="alert" className="text-red-500 text-sm mt-1">
              {errors.email.message}
            </p>
          )}
        </div>
        <div className="flex flex-col items-start">
          <input
            id="password"
            type="password"
            {...register("password")}
            placeholder="********"
            aria-invalid={!!errors.password}
            aria-describedby={errors.password ? "password-error" : undefined}
            className={`w-full p-1.5 rounded-md text-xs text-text font-semibold border bg-input-bg placeholder:text-clock-black-d ${
              errors.password
                ? "border-red-500 hover:border-red-400"
                : "border-input-border hover:border-gray-500"
            }`}
          />
          {errors.password && (
            <p
              id="password-error"
              role="alert"
              className="text-red-500 text-sm mt-1"
            >
              {errors.password.message}
            </p>
          )}
        </div>
        {/* onSubmit sets a root error on a failed login; without this it
            was set and never rendered. */}
        {errors.root && (
          <p role="alert" className="text-red-500 text-sm">
            {errors.root.message}
          </p>
        )}
        <button
          type="submit"
          className="w-full p-2 text-text text-md font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-accent to-accent-strong hover:bg-linear-to-b hover:from-accent-hover hover:to-accent-strong"
        >
          Login
        </button>
      </form>
    </div>
  );
}
