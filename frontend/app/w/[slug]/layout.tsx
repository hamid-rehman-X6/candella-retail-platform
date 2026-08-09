import type { Metadata } from "next";
import { redirect, notFound } from "next/navigation";
import { getServerUser, getServerWorkspaces } from "@/lib/server-auth";
import { AppShell } from "@/components/app/app-shell";

export const metadata: Metadata = { title: "Dashboard" };

/**
 * Server-side guard for one workspace. Runs before any UI renders:
 * not signed in → /login; no workspaces at all → /onboarding; a slug the user
 * isn't a member of → 404 (never 403, which would confirm the workspace exists).
 *
 * The workspace lives in the URL rather than in client state, so a link is
 * unambiguous and two tabs can sit on two different workspaces safely.
 */
export default async function WorkspaceLayout({
  params,
  children,
}: {
  params: Promise<{ slug: string }>;
  children: React.ReactNode;
}) {
  const { slug } = await params;

  const user = await getServerUser();
  if (!user) redirect("/login");

  const workspaces = await getServerWorkspaces();
  if (workspaces.length === 0) redirect("/onboarding");

  const current = workspaces.find((w) => w.slug === slug);
  if (!current) notFound();

  return (
    <AppShell user={user} workspaces={workspaces} current={current}>
      {children}
    </AppShell>
  );
}
