import type { Metadata } from "next";
import { VerifyEmailForm } from "@/components/auth/forms/verify-email-form";

export const metadata: Metadata = { title: "Verify email" };

export default function VerifyEmailPage() {
  return <VerifyEmailForm />;
}
