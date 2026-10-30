import { registrationFormSchema, type RegistrationFormData } from "../../data/authSchema";
import { zodResolver } from "@hookform/resolvers/zod";
import { registerUser } from "../../lib/services/user";
import { Link, useNavigate } from "react-router";
import { useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { AuthLayout, ProviderButtons } from "./auth-layout";
import { TextField } from "../../components/ui/text-field";
import { Button } from "../../components/ui/button";

// The signup rating seeds all three speeds (Bullet, Blitz, Rapid).
const SKILL_LEVELS = [
  { rating: 400, label: "New to chess", detail: "I don't know how to play" },
  { rating: 800, label: "Beginner", detail: "I know the rules and basics" },
  { rating: 1200, label: "Intermediate", detail: "I know strategies and tactics" },
  { rating: 1600, label: "Advanced", detail: "I'm a tournament player" },
] as const;

const STEPS = ["Skill level", "Email and password", "Username"] as const;

/**
 * Three steps, one field group each: skill level, then email and password,
 * then username. Each Continue validates only its own step's fields; the
 * values survive moving back and forth because react-hook-form keeps
 * unmounted fields' values by default.
 */
export default function Signup() {
  const [step, setStep] = useState(0);
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    setError,
    trigger,
    control,
    formState: { errors, isSubmitting },
  } = useForm<RegistrationFormData>({
    resolver: zodResolver(registrationFormSchema),
    mode: "onTouched",
  });
  const rating = useWatch({ control, name: "rating" });

  const continueFromCredentials = async () => {
    if (await trigger(["email", "password"])) setStep(2);
  };

  const onSubmit = async (data: RegistrationFormData) => {
    try {
      await registerUser(data);
      navigate("/");
    } catch (err) {
      // The server says which field is taken; go back to its step.
      const message = err instanceof Error ? err.message : "";
      if (/email/i.test(message)) {
        setStep(1);
        setError("email", { message: "That email is already registered." });
      } else if (/username/i.test(message)) {
        setError("username", { message: "That username is taken." });
      } else {
        setError("root", { message: "Something went wrong. Please try again." });
      }
    }
  };

  return (
    <AuthLayout
      footer={
        <>
          Already have an account?{" "}
          <Link to="/login" className="font-extrabold text-accent hover:text-accent-hover">
            Log In
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="flex flex-col gap-4 p-8">
        <div className="flex items-center justify-between">
          {step > 0 ? (
            <Button variant="ghost" size="sm" onClick={() => setStep(step - 1)} className="-ml-3">
              ← Back
            </Button>
          ) : (
            <span />
          )}
          <ol aria-label={`Step ${step + 1} of ${STEPS.length}`} className="flex gap-1.5">
            {STEPS.map((name, i) => (
              <li
                key={name}
                aria-current={i === step ? "step" : undefined}
                className={`h-1.5 w-8 rounded-full ${i <= step ? "bg-accent" : "bg-options-button"}`}
              />
            ))}
          </ol>
        </div>

        {step === 0 && (
          <>
            <h1 className="text-2xl font-extrabold">What is your chess skill level?</h1>
            <Controller
              name="rating"
              control={control}
              render={({ field }) => (
                <div role="radiogroup" aria-label="Skill level" className="flex flex-col gap-2">
                  {SKILL_LEVELS.map((level) => {
                    const checked = field.value === level.rating;
                    return (
                      <label
                        key={level.rating}
                        className={`flex items-center gap-4 rounded-lg border-2 p-4 text-left hover:cursor-pointer hover:bg-options-button ${
                          checked ? "border-accent bg-options-button" : "border-transparent bg-input-bg"
                        }`}
                      >
                        <input
                          type="radio"
                          name={field.name}
                          value={level.rating}
                          checked={checked}
                          onChange={() => field.onChange(level.rating)}
                          className="sr-only"
                        />
                        <span className="flex flex-col">
                          <span className="font-extrabold">{level.label}</span>
                          <span className="text-sm text-clock-disabled">{level.detail}</span>
                        </span>
                      </label>
                    );
                  })}
                </div>
              )}
            />
            <Button size="lg" disabled={rating === undefined} onClick={() => setStep(1)} className="w-full">
              Continue
            </Button>
          </>
        )}

        {step === 1 && (
          <>
            <h1 className="text-2xl font-extrabold">Enter your email and a password</h1>
            <TextField id="email" type="email" label="Email" placeholder="Email" autoComplete="email" error={errors.email?.message} {...register("email")} />
            <TextField
              id="password"
              type="password"
              label="Password"
              placeholder="Password"
              autoComplete="new-password"
              error={errors.password?.message}
              {...register("password")}
            />
            <Button size="lg" onClick={continueFromCredentials} className="w-full">
              Continue
            </Button>
            <ProviderButtons providers={["Google", "Apple"]} />
          </>
        )}

        {step === 2 && (
          <>
            <h1 className="text-2xl font-extrabold">Choose a username</h1>
            <p className="-mt-2 text-left text-sm text-clock-disabled">This is what other players will see.</p>
            <TextField id="username" label="Username" placeholder="Username" autoComplete="username" error={errors.username?.message} {...register("username")} />
            {errors.root && (
              <p role="alert" className="text-left text-sm text-danger">
                {errors.root.message}
              </p>
            )}
            <Button type="submit" size="lg" disabled={isSubmitting} className="w-full">
              Create Account
            </Button>
          </>
        )}
      </form>
    </AuthLayout>
  );
}
