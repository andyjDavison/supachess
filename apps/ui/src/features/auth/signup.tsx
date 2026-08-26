import {
  registrationFormSchema,
  type RegistrationFormData,
} from "../../data/authSchema";
import { zodResolver } from "@hookform/resolvers/zod";
import { registerUser } from "../../lib/services/user";
import { Link, useNavigate } from "react-router";
import { useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";

export default function Signup() {
  const [formSection, setFormSection] = useState(0);
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    setError,
    control,
    formState: { errors, isSubmitting },
  } = useForm<RegistrationFormData>({
    resolver: zodResolver(registrationFormSchema),
    mode: "onBlur",
  });

  useEffect(() => {}, [formSection]);

  const onSubmit = async (data: RegistrationFormData) => {
    try {
      const res = await registerUser(data);

      if (!res.ok) {
        const { message } = await res.json();
        setError("root", {
          message: message ?? "Something went wrong. Please try again.",
        });
        return;
      }
      navigate("/");
    } catch {
      setError("root", { message: "Network error. Please try again." });
    }
  };

  return (
    <div className="flex flex-col gap-4 pt-4 items-center">
      {formSection === 0 && (
        <>
          <header>
            <Link to="/" className="text-white/90 text-2xl font-extrabold">
              Supachess.com!
            </Link>
          </header>
          <div className="flex flex-col items-center justify-between h-140 w-40/50">
            <h1 className="text-3xl text-white/90 font-extrabold">
              Create Your Supachess.com Account
            </h1>
            <div className="flex flex-col w-full items-center gap-2">
              <button
                onClick={() => setFormSection(1)}
                className="w-4/5 p-4 text-white/90 text-xl font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-fuchsia-400 to-fuchsia-500 hover:bg-linear-to-b hover:from-fuchsia-300 hover:to-fuchsia-500"
              >
                Contine with Email
              </button>
              <p className="text-xs m-3">OR</p>
              <button className="w-4/5 p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end">
                Contine with Phone
              </button>
              <button className="w-4/5 p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end">
                Contine with Google
              </button>
              <button className="w-4/5 p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end">
                Contine with Apple
              </button>
            </div>
          </div>
        </>
      )}
      {formSection === 1 && (
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
          <div className="flex flex-col items-start">
            <input
              id="username"
              type="text"
              {...register("username")}
              placeholder="Username"
              className={`w-full p-1.5 rounded-md text-xs text-white/50 font-semibold border bg-input-bg ${
                errors.email
                  ? "border-red-500 hover:border-red-400"
                  : "border-input-border hover:border-gray-500"
              }`}
            />
            {errors.username && (
              <p
                id="email-error"
                role="alert"
                className="text-red-500 text-sm mt-1"
              >
                {errors.username.message}
              </p>
            )}
          </div>
          <div className="flex flex-col gap-2 w-full">
            <Controller
              name="rating"
              control={control}
              render={({ field }) => {
                const isChecked = field.value === 400;

                return (
                  <label
                    className={`w-full p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-fuchsia-400" : ""}
                      ${errors.rating ? "border border-red-500" : ""}`}
                  >
                    <input
                      type="radio"
                      name={field.name}
                      checked={field.value === 400}
                      onChange={() => field.onChange(400)}
                      value={400}
                      className="peer sr-only"
                    />
                    <span>I don't know how to play</span>
                  </label>
                );
              }}
            />
            <Controller
              name="rating"
              control={control}
              render={({ field }) => {
                const isChecked = field.value === 800;
                return (
                  <label
                    className={`w-full p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-fuchsia-400" : ""}
                      ${errors.rating ? "border border-red-500" : ""}`}
                  >
                    <input
                      type="radio"
                      name={field.name}
                      checked={isChecked}
                      onChange={() => field.onChange(800)}
                      value={800}
                      className="peer sr-only"
                    />
                    <span>I know the rules and basics</span>
                  </label>
                );
              }}
            />
            <Controller
              name="rating"
              control={control}
              render={({ field }) => {
                const isChecked = field.value === 1200;
                return (
                  <label
                    className={`w-full p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-fuchsia-400" : ""}
                      ${errors.rating ? "border border-red-500" : ""}`}
                  >
                    <input
                      type="radio"
                      name={field.name}
                      checked={isChecked}
                      onChange={() => field.onChange(1200)}
                      value={1200}
                      className="peer sr-only"
                    />
                    <span>I know strategies and tactics</span>
                  </label>
                );
              }}
            />
            <Controller
              name="rating"
              control={control}
              render={({ field }) => {
                const isChecked = field.value === 1600;
                return (
                  <label
                    className={`w-full p-3 text-white/90 text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-fuchsia-400" : ""}
                      ${errors.rating ? "border border-red-500" : ""}`}
                  >
                    <input
                      type="radio"
                      name={field.name}
                      checked={isChecked}
                      onChange={() => field.onChange(1600)}
                      value={1600}
                      className="peer sr-only"
                    />
                    <span>I'm a tournament player</span>
                  </label>
                );
              }}
            />
          </div>
          <button
            type="submit"
            className="w-full p-4 text-white/90 text-xl font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-fuchsia-400 to-fuchsia-500 hover:bg-linear-to-b hover:from-fuchsia-300 hover:to-fuchsia-500"
          >
            Register
          </button>
        </form>
      )}
    </div>
  );
}
