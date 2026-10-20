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
    formState: { errors },
  } = useForm<RegistrationFormData>({
    resolver: zodResolver(registrationFormSchema),
    mode: "onBlur",
  });

  useEffect(() => {}, [formSection]);

  const onSubmit = async (data: RegistrationFormData) => {
    try {
      await registerUser(data);
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
            <Link to="/" className="text-text text-2xl font-extrabold">
              Supachess.com!
            </Link>
          </header>
          <div className="flex flex-col items-center justify-between h-140 w-40/50">
            <h1 className="text-3xl text-text font-extrabold">
              Create Your Supachess.com Account
            </h1>
            <div className="flex flex-col w-full items-center gap-2">
              <button
                onClick={() => setFormSection(1)}
                className="w-4/5 p-4 text-text text-xl font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-accent to-accent-strong hover:bg-linear-to-b hover:from-accent-hover hover:to-accent-strong"
              >
                Continue with Email
              </button>
              <p className="text-xs m-3">OR</p>
              <button
                disabled
                className="w-4/5 p-3 text-text text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end disabled:cursor-not-allowed disabled:text-clock-black-d"
              >
                Continue with Phone
              </button>
              <button
                disabled
                className="w-4/5 p-3 text-text text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end disabled:cursor-not-allowed disabled:text-clock-black-d"
              >
                Continue with Google
              </button>
              <button
                disabled
                className="w-4/5 p-3 text-text text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end disabled:cursor-not-allowed disabled:text-clock-black-d"
              >
                Continue with Apple
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
              aria-invalid={!!errors.email}
              aria-describedby={errors.email ? "email-error" : undefined}
              className={`w-full p-1.5 rounded-md text-xs text-text font-semibold border bg-input-bg placeholder:text-clock-black-d ${
                errors.email
                  ? "border-red-500 hover:border-red-400"
                  : "border-input-border hover:border-gray-500"
              }`}
            />
            {errors.email && (
              <p
                id="password-error"
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
                id="username-error"
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
              aria-invalid={!!errors.username}
              aria-describedby={errors.username ? "username-error" : undefined}
              className={`w-full p-1.5 rounded-md text-xs text-text font-semibold border bg-input-bg placeholder:text-clock-black-d ${
                errors.username
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
                    className={`w-full p-3 text-text text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-accent" : ""}
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
                    className={`w-full p-3 text-text text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-accent" : ""}
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
                    className={`w-full p-3 text-text text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-accent" : ""}
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
                    className={`w-full p-3 text-text text-lg font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-gradient-start to-gradient-end hover:bg-linear-to-b hover:from-hover-gradient-start hover:to-hover-gradient-end 
                      ${isChecked ? "border border-accent" : ""}
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
            className="w-full p-4 text-text text-xl font-extrabold rounded-lg hover:cursor-pointer bg-linear-to-b from-accent to-accent-strong hover:bg-linear-to-b hover:from-accent-hover hover:to-accent-strong"
          >
            Register
          </button>
        </form>
      )}
    </div>
  );
}
