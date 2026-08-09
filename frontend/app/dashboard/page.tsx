import { redirect } from "next/navigation";
import { getServerUser, getServerWorkspaces } from "@/lib/server-auth";

/**
 * "Take me to my workspace." Sign-in, OAuth and any old bookmark land here, and
 * this resolves the user's default workspace and forwards to its real URL —
 * keeping every entry point pointing at one stable path while the app itself
 * stays workspace-addressed.
 */
export default async function DashboardPage() {
  const user = await getServerUser();
  if (!user) redirect("/login");

  const workspaces = await getServerWorkspaces();
  const first = workspaces[0];
  if (!first) redirect("/onboarding");

  redirect(`/w/${first.slug}`);
}
