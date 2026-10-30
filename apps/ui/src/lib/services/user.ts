import type { RegistrationFormData } from "../../data/authSchema";

const API_URL = import.meta.env.VITE_API_URL;

export async function registerUser(data: RegistrationFormData) {
  const res = await fetch(`${API_URL}/api/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  // Errors come back as plain text ("username already taken"), which the
  // signup form uses to send you back to the step with that field.
  const text = await res.text();
  if (!res.ok) {
    throw new Error(text.trim() || "Registration failed");
  }

  return JSON.parse(text);
}

export async function checkEmailAvailable(email: string): Promise<boolean> {
  const res = await fetch(
    `${API_URL}/api/user/email/${encodeURIComponent(email)}`,
  );
  return res.status === 404;
}

export async function checkUsernameAvailable(
  username: string,
): Promise<boolean> {
  const res = await fetch(
    `${API_URL}/api/user/name/${encodeURIComponent(username)}`,
  );
  return res.status === 404;
}
