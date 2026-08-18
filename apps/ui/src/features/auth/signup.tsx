import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { registrationFormSchema, type RegistrationFormData } from "./schemas";
import { zodResolver } from "@hookform/resolvers/zod";
import { registerUser } from "../../lib/services/user";

function Signup() {
  const [formSection, setFormSection] = useState(0);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegistrationFormData>({
    resolver: zodResolver(registrationFormSchema),
  });

  useEffect(() => {}, [formSection]);

  const onSubmit = async (data: RegistrationFormData) => {
    try {
      const user = await registerUser(data);
    } catch (err) {
      console.log(err);
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
          <div>
            <input
              id="username"
              type="text"
              {...register("username")}
              placeholder="Username"
              className="w-full p-1.5 rounded-md text-xs text-white/50 font-semibold border bg-input-bg border-input-border hover:border-gray-500"
            />
          </div>
          <div>
            <input
              id="email"
              type="text"
              {...register("email")}
              placeholder="Email"
              className="w-full p-1.5 rounded-md text-xs text-white/50 font-semibold border bg-input-bg border-input-border hover:border-gray-500"
            />
          </div>
          <div>
            <input
              id="password"
              type="text"
              {...register("password")}
              placeholder="********"
              className="w-full p-1.5 rounded-md text-xs text-white/50 font-semibold border bg-input-bg border-input-border hover:border-gray-500"
            />
          </div>
        </form>
      )}
    </div>
  );
}

export default Signup;
