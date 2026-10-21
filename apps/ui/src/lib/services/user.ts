import type { RegistrationFormData } from "../../data/authSchema";

const API_URL = import.meta.env.VITE_API_URL;

export async function registerUser(data: RegistrationFormData) {
  const res = await fetch(`${API_URL}/api/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  const body = await res.json().catch(() => null);

  if (!res.ok) {
    throw new Error(body?.message ?? "Registration failed");
  }

  return body;
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
