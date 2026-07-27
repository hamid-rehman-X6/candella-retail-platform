import type { Metadata } from "next";
import { TwoFactorForm } from "@/components/auth/forms/two-factor-form";

export const metadata: Metadata = { title: "Two-factor authentication" };

export default function TwoFactorPage() {
  return <TwoFactorForm />;
}
