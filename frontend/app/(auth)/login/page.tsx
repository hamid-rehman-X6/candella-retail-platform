import type { Metadata } from "next";
import { LoginForm } from "@/components/auth/forms/login-form";

export const metadata: Metadata = { title: "Sign in" };

export default function LoginPage() {
  return <LoginForm />;
}
