import { BrandPanel } from "@/components/auth/brand-panel";
import { Logo } from "@/components/ui/logo";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-canvas lg:grid lg:grid-cols-[1.05fr_1fr]">
      <BrandPanel />

      <div className="relative flex min-h-screen flex-col">
        {/* soft top glow on the form side for cohesion with the brand panel */}
        <div
          aria-hidden
          className="pointer-events-none absolute inset-x-0 top-0 h-64 bg-[radial-gradient(60%_100%_at_50%_0%,rgba(245,158,11,0.1),transparent_70%)]"
        />

        {/* mobile-only logo header (brand panel is hidden on small screens) */}
        <header className="relative z-10 flex items-center p-6 lg:hidden">
          <Logo />
        </header>

        <main className="relative z-10 flex flex-1 items-center justify-center px-6 pt-4 pb-12 sm:px-10">
          <div className="w-full max-w-[26rem]">{children}</div>
        </main>
      </div>
    </div>
  );
}
