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
            className={`w-full p-1.5 rounded-md text-xs text-white/50 font-semibold border bg-input-bg ${
              errors.email
                ? "border-red-500 hover:border-red-400"
                : "border-input-border hover:border-gray-500"
            }`}
          />
          {errors.email && (
            <p
              id="email-error"
              role="alert"
              className="text-red-500 text-sm mt-1"
            >
              {errors.email.message}
            </p>
          )}
        </div>
        <div className="flex flex-col items-start">
          <input
            id="password"
            type="text"
            {...register("password")}
            placeholder="********"
            className={`w-full p-1.5 rounded-md text-xs text-white/50 font-semibold border bg-input-bg ${
              errors.email
                ? "border-red-500 hover:border-red-400"
                : "border-input-border hover:border-gray-500"
            }`}
          />
          {errors.password && (
            <p
              id="email-error"
              role="alert"
              className="text-red-500 text-sm mt-1"
            >
              {errors.password.message}
            </p>
          )}
        </div>
        <button
          type="submit"
          className="w-full p-2 text-white/90 text-md font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-fuchsia-400 to-fuchsia-500 hover:bg-linear-to-b hover:from-fuchsia-300 hover:to-fuchsia-500"
        >
          Login
        </button>
      </form>
    </div>
  );
}
