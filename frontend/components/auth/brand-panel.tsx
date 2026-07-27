import { Check, Wifi, RefreshCw, ShieldCheck } from "lucide-react";
import { Logo } from "@/components/ui/logo";

const points = [
  { icon: Wifi, text: "Offline-first POS that never stops selling" },
  { icon: RefreshCw, text: "Real-time stock across every store" },
  { icon: ShieldCheck, text: "Enterprise-grade security, built in" },
];

/** Left showcase panel of the auth split layout (desktop only). */
export function BrandPanel() {
  return (
    <div className="relative hidden overflow-hidden border-r border-line bg-canvas-raised lg:flex lg:flex-col">
      {/* ambient */}
      <div className="bg-grid absolute inset-0 opacity-40" />
      <div className="absolute -top-32 -left-24 h-[32rem] w-[32rem] animate-aurora rounded-full bg-[radial-gradient(circle,rgba(245,158,11,0.22),transparent_62%)] blur-3xl" />
      <div className="absolute right-[-10%] bottom-[-10%] h-[28rem] w-[28rem] animate-aurora rounded-full bg-[radial-gradient(circle,rgba(180,83,9,0.18),transparent_65%)] blur-3xl [animation-delay:-8s]" />
      <div className="absolute inset-0 bg-[radial-gradient(120%_80%_at_50%_-10%,transparent_45%,rgba(0,0,0,0.5))]" />

      <div className="relative flex h-full flex-col justify-between p-12">
        <Logo />

        <div className="max-w-md">
          <h2 className="text-4xl leading-[1.1] font-semibold tracking-tight">
            Run your whole retail business on{" "}
            <span className="text-gradient">one platform.</span>
          </h2>
          <p className="mt-5 text-base leading-relaxed text-muted">
            POS, inventory, pharmacy, garments, cosmetics, CRM and analytics —
            unified, so nothing ever falls out of sync.
          </p>

          <ul className="mt-8 flex flex-col gap-3.5">
            {points.map((p) => {
              const Icon = p.icon;
              return (
                <li key={p.text} className="flex items-center gap-3">
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-accent/10 text-accent ring-1 ring-accent/20">
                    <Icon className="h-4 w-4" />
                  </span>
                  <span className="text-sm text-foreground/85">{p.text}</span>
                </li>
              );
            })}
          </ul>
        </div>

        <figure className="glass max-w-md rounded-2xl p-5">
          <blockquote className="text-sm leading-relaxed text-foreground/90">
            “We replaced four disconnected tools with Candella. Stock, sales and
            customers finally live in one place.”
          </blockquote>
          <figcaption className="mt-3 flex items-center gap-2 text-xs text-muted">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-accent/15 text-[10px] font-semibold text-accent">
              AK
            </span>
            Ayesha Khan · MediCare Pharmacy Group
          </figcaption>
        </figure>
      </div>
    </div>
  );
}

/** Small reusable "included" list item, used on confirmation screens. */
export function CheckLine({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-2.5 text-sm text-foreground/85">
      <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-accent/15 text-accent">
        <Check className="h-3 w-3" />
      </span>
      {children}
    </li>
  );
}
