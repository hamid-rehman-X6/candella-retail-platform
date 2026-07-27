import { Container } from "@/components/ui/container";
import { Logo } from "@/components/ui/logo";
import { footerColumns } from "@/lib/site";

export function Footer() {
  return (
    <footer className="relative overflow-hidden border-t border-line pt-20">
      {/* Layered, lightly-lit background for depth */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 overflow-hidden"
      >
        {/* lift from the dark canvas to a lighter tone toward the bottom */}
        <div className="absolute inset-0 bg-linear-to-b from-transparent via-canvas-raised to-surface" />

        {/* fading grid texture */}
        <div className="bg-grid absolute inset-0 [mask-image:linear-gradient(to_bottom,transparent,#000_55%)] opacity-60" />

        {/* animated amber aurora rising from the bottom */}
        <div className="absolute -bottom-48 left-1/2 h-[40rem] w-[80rem] -translate-x-1/2 animate-aurora rounded-full bg-[radial-gradient(circle,rgba(245,158,11,0.16),transparent_60%)] blur-3xl" />
        <div className="absolute bottom-0 left-[8%] h-72 w-72 animate-float-slow rounded-full bg-[radial-gradient(circle,rgba(251,191,36,0.1),transparent_65%)] blur-3xl" />
        <div className="absolute right-[6%] bottom-10 h-64 w-64 animate-float rounded-full bg-[radial-gradient(circle,rgba(180,83,9,0.12),transparent_65%)] blur-3xl [animation-delay:-4s]" />

        {/* large brand watermark, fading out downward */}
        <span className="absolute inset-x-0 -bottom-4 bg-linear-to-b from-white/[0.055] to-transparent bg-clip-text text-center text-[19vw] leading-none font-bold tracking-tighter whitespace-nowrap text-transparent select-none">
          Candella
        </span>

        {/* top accent hairline */}
        <div className="absolute inset-x-0 top-0 h-px bg-linear-to-r from-transparent via-accent/50 to-transparent" />
      </div>

      <Container>
        <div className="grid gap-12 pb-14 lg:grid-cols-[1.5fr_2fr]">
          {/* brand */}
          <div className="max-w-sm">
            <Logo />
            <p className="mt-5 text-sm leading-relaxed text-muted">
              The unified retail operating system. POS, inventory, pharmacy,
              garments, cosmetics, CRM and analytics — one platform for every
              retail business.
            </p>
            <p className="mt-6 text-sm font-medium text-foreground/80">
              One platform, every retail business.
            </p>
          </div>

          {/* link columns */}
          <div className="grid grid-cols-2 gap-8 sm:grid-cols-4">
            {footerColumns.map((col) => (
              <div key={col.title}>
                <h4 className="text-sm font-semibold tracking-tight text-foreground">
                  {col.title}
                </h4>
                <ul className="mt-4 space-y-3">
                  {col.links.map((link) => (
                    <li key={link.label}>
                      <a
                        href={link.href}
                        className="text-sm text-muted transition-colors hover:text-foreground"
                      >
                        {link.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <div className="flex flex-col items-center justify-between gap-4 border-t border-line py-8 sm:flex-row">
          <p className="text-sm text-subtle">
            © {new Date().getFullYear()} Candella. All rights reserved.
          </p>
          <div className="flex gap-6 text-sm text-subtle">
            <a href="#" className="transition-colors hover:text-foreground">
              Privacy
            </a>
            <a href="#" className="transition-colors hover:text-foreground">
              Terms
            </a>
            <a href="#" className="transition-colors hover:text-foreground">
              Security
            </a>
          </div>
        </div>
      </Container>
    </footer>
  );
}
