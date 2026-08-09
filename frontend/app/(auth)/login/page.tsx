import type { Metadata } from "next";
import { LoginForm } from "@/components/auth/forms/login-form";

export const metadata: Metadata = { title: "Sign in" };

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  // Redirect flows (Google) report failures by sending the user back here with
  // an ?error= code — surfacing it is what turns a dead end into an explanation.
  const raw = (await searchParams).error;
  const errorCode = Array.isArray(raw) ? raw[0] : raw;

  return <LoginForm errorCode={errorCode} />;
}
