import { z } from "zod";

export const registrationFormSchema = z.object({
  username: z.string().min(3, "Username must be at least 3 characters"),
  email: z.email("Enter a valid email address"),
  password: z.string().min(8, "Password msut be at least 8 characters"),
  rating: z.int().gte(0, "Select your approximate skill level"),
});

export type RegistrationFormData = z.infer<typeof registrationFormSchema>;
