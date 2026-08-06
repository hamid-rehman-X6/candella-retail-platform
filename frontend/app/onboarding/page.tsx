import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { getServerUser } from "@/lib/server-auth";
import { BackgroundEffects } from "@/components/landing/background-effects";
import { OnboardingForm } from "@/components/app/onboarding-form";

export const metadata: Metadata = { title: "Create your workspace" };

export default async function OnboardingPage() {
  const user = await getServerUser();
  if (!user) redirect("/login");

  // Users who already have a workspace can still reach here (via the switcher's
  // "New workspace"), so we intentionally don't redirect them away.
  return (
    <>
      <BackgroundEffects />
      <main className="relative flex min-h-screen items-center justify-center px-6 py-16">
        <OnboardingForm userName={user.fullName} />
      </main>
    </>
  );
}
