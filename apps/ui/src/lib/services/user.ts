import type { RegistrationFormData } from "../../features/auth/schemas";

const API_URL = "http://localhost:8080/";

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
