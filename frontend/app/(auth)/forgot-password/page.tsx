import type { Metadata } from "next";
import { ForgotPasswordForm } from "@/components/auth/forms/forgot-password-form";

export const metadata: Metadata = { title: "Forgot password" };

export default function ForgotPasswordPage() {
  return <ForgotPasswordForm />;
}
