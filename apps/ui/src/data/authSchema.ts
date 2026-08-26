import { z } from "zod";

export const registrationFormSchema = z.object({
  username: z.string().min(3, "Username must be at least 3 characters"),
  email: z.email("Enter a valid email address"),
  password: z.string().min(8, "Password msut be at least 8 characters"),
  rating: z.int().gte(0, "Select your approximate skill level"),
});

export const loginFormSchema = z.object({
  email: z.email("Enter a valid email address"),
  password: z.string().min(1, "Password is required"),
});

type LoginFormData = z.infer<typeof loginFormSchema>;
type RegistrationFormData = z.infer<typeof registrationFormSchema>;

export { type LoginFormData, type RegistrationFormData };
