import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { getServerUser, getServerWorkspaces } from "@/lib/server-auth";
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
  const params = await searchParams;
  const welcome = params.welcome;
  const signedInWith = welcome === "google" ? user.email : undefined;

  // Resolve the requested workspace against the signed-in user's memberships.
  // A crafted query string can never turn this into an external redirect.
  const workspaces = await getServerWorkspaces();
  const from = typeof params.from === "string" ? params.from : "";
  const previous = workspaces.find((workspace) => workspace.slug === from);
  const returnHref = previous
    ? `/w/${previous.slug}`
    : workspaces.length > 0
      ? "/dashboard"
      : "/";
  const returnLabel = previous
    ? `Back to ${previous.name}`
    : workspaces.length > 0
      ? "Back to your workspace"
      : "Back to home";

  // Users who already have a workspace can still reach here (via the switcher's
  // "New workspace"), so we intentionally don't redirect them away.
  return (
    <>
      <BackgroundEffects />
      <main className="relative flex min-h-screen items-center justify-center px-6 py-16">
        <OnboardingForm
          userName={user.fullName}
          signedInWith={signedInWith}
          returnHref={returnHref}
          returnLabel={returnLabel}
        />
      </main>
    </>
  );
}
