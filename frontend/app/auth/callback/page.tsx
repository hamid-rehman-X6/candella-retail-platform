import type { Metadata } from "next";
import { BackgroundEffects } from "@/components/landing/background-effects";
import { OAuthCallback } from "@/components/auth/oauth-callback";

export const metadata: Metadata = {
  title: "Signing you in",
  robots: { index: false },
};

/** A query param can arrive repeated; we only ever care about the first value. */
function first(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

export default async function AuthCallbackPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;

  return (
    <>
      <BackgroundEffects />
      <main className="relative flex min-h-screen items-center justify-center px-6">
        <OAuthCallback
          status={first(params.status) ?? "error"}
          event={first(params.event)}
          reason={first(params.reason)}
          mfaChallenge={first(params.mfaToken)}
        />
      </main>
    </>
  );
}
