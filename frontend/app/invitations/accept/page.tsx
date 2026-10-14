import { InvitationAcceptance } from "./invitation-acceptance";

export const metadata = { title: "Accept staff invitation" };

export default async function AcceptInvitationPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string | string[] }>;
}) {
  const raw = (await searchParams).token;
  const token = (Array.isArray(raw) ? raw[0] : raw) ?? "";
  return <InvitationAcceptance token={token} />;
}
