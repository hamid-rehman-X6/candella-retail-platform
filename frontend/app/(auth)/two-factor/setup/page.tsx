import type { Metadata } from "next";
import { TwoFactorSetupForm } from "@/components/auth/forms/two-factor-setup-form";

export const metadata: Metadata = { title: "Set up two-factor auth" };

export default function TwoFactorSetupPage() {
  return <TwoFactorSetupForm />;
}
