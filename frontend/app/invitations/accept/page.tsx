import { InvitationAcceptance } from "./invitation-acceptance";
import { ThemeToggle } from "@/components/ui/theme-toggle";

export const metadata = { title: "Accept staff invitation" };

export default async function AcceptInvitationPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string | string[] }>;
}) {
  const raw = (await searchParams).token;
  const token = (Array.isArray(raw) ? raw[0] : raw) ?? "";
  return (
    <>
      <div className="fixed top-5 right-5 z-20">
        <ThemeToggle />
      </div>
      <InvitationAcceptance token={token} />
    </>
  );
}
