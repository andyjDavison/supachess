import { loginFormSchema, registrationFormSchema } from "./authSchema";

function errorFor(
  schema: typeof registrationFormSchema | typeof loginFormSchema,
  input: unknown,
  field: string,
): string | undefined {
  const result = schema.safeParse(input);
  if (result.success) return undefined;
  return result.error.issues.find((i) => i.path[0] === field)?.message;
}

const validRegistration = {
  username: "alice",
  email: "alice@example.com",
  password: "password123",
  rating: 800,
};

describe("registrationFormSchema", () => {
  it("accepts a complete valid registration", () => {
    expect(registrationFormSchema.safeParse(validRegistration).success).toBe(true);
  });

  describe("username", () => {
    it("requires at least three characters", () => {
      expect(errorFor(registrationFormSchema, { ...validRegistration, username: "ab" }, "username")).toBe(
        "Username must be at least 3 characters",
      );
    });

    it("accepts exactly three", () => {
      expect(
        registrationFormSchema.safeParse({ ...validRegistration, username: "abc" }).success,
      ).toBe(true);
    });
  });

  describe("password", () => {
    it("requires at least eight characters", () => {
      expect(errorFor(registrationFormSchema, { ...validRegistration, password: "1234567" }, "password")).toBe(
        "Password must be at least 8 characters",
      );
    });

    it("accepts exactly eight", () => {
      expect(
        registrationFormSchema.safeParse({ ...validRegistration, password: "12345678" }).success,
      ).toBe(true);
    });
  });

  describe("email", () => {
    it.each(["no-at-sign", "", "@nodomain", "spaces in@email.com"])(
      "rejects %j",
      (email) => {
        expect(registrationFormSchema.safeParse({ ...validRegistration, email }).success).toBe(false);
      },
    );

    it("reports the friendly message", () => {
      expect(errorFor(registrationFormSchema, { ...validRegistration, email: "nope" }, "email")).toBe(
        "Enter a valid email address",
      );
    });
  });

  describe("rating", () => {
    // The signup form supplies this from the skill-level radios, so a
    // missing selection has to fail rather than default to zero.
    it("is required", () => {
      const { rating, ...withoutRating } = validRegistration;
      void rating;
      expect(registrationFormSchema.safeParse(withoutRating).success).toBe(false);
    });

    it("rejects negatives", () => {
      expect(registrationFormSchema.safeParse({ ...validRegistration, rating: -1 }).success).toBe(false);
    });

    it("rejects a non-integer", () => {
      expect(registrationFormSchema.safeParse({ ...validRegistration, rating: 800.5 }).success).toBe(false);
    });

    it.each([400, 800, 1200, 1600])("accepts the %i preset", (rating) => {
      expect(registrationFormSchema.safeParse({ ...validRegistration, rating }).success).toBe(true);
    });
  });
});

describe("loginFormSchema", () => {
  it("accepts an email and any non-empty password", () => {
    expect(loginFormSchema.safeParse({ email: "a@b.com", password: "x" }).success).toBe(true);
  });

  // Login must not enforce the registration minimum: an existing account
  // could predate the rule, and the server is the one that decides.
  it("does not apply the eight-character minimum", () => {
    expect(loginFormSchema.safeParse({ email: "a@b.com", password: "short" }).success).toBe(true);
  });

  it("requires a password", () => {
    expect(errorFor(loginFormSchema, { email: "a@b.com", password: "" }, "password")).toBe(
      "Password is required",
    );
  });

  it("requires a valid email", () => {
    expect(loginFormSchema.safeParse({ email: "nope", password: "x" }).success).toBe(false);
  });
});
