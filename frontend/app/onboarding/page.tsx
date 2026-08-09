import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { getServerUser } from "@/lib/server-auth";
import { BackgroundEffects } from "@/components/landing/background-effects";
import { OnboardingForm } from "@/components/app/onboarding-form";

export const metadata: Metadata = { title: "Create your workspace" };

export default async function OnboardingPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const user = await getServerUser();
  if (!user) redirect("/login");

  // Set by the OAuth bridge page after a Google sign-up, so the workspace form
  // reads as step 2 of 2 rather than a screen that appeared out of nowhere.
  const welcome = (await searchParams).welcome;
  const signedInWith = welcome === "google" ? user.email : undefined;

  // Users who already have a workspace can still reach here (via the switcher's
  // "New workspace"), so we intentionally don't redirect them away.
  return (
    <>
      <BackgroundEffects />
      <main className="relative flex min-h-screen items-center justify-center px-6 py-16">
        <OnboardingForm userName={user.fullName} signedInWith={signedInWith} />
      </main>
    </>
  );
}
