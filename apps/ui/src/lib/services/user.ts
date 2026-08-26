import type { RegistrationFormData } from "../../data/authSchema";

const API_URL = "http://localhost:8080";

export async function registerUser(data: RegistrationFormData) {
  const res = await fetch(`${API_URL}/api/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  const body = await res.json();

  try {
    return body;
  } catch (err) {
    console.log("some error");
  }
}

export async function checkEmailAvailable(email: string): Promise<boolean> {
  const res = await fetch(
    `${API_URL}/api/users/email/${encodeURIComponent(email)}`,
  );
  return res.status === 404;
}

export async function checkUsernameAvailable(
  username: string,
): Promise<boolean> {
  const res = await fetch(
    `${API_URL}/api/users/name/${encodeURIComponent(username)}`,
  );
  return res.status === 404;
}
