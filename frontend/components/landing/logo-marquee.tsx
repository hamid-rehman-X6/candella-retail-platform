import { Container } from "@/components/ui/container";
import { marqueeLogos } from "@/lib/site";

/** Infinite auto-scrolling row of customer wordmarks (CSS marquee). */
export function LogoMarquee() {
  const row = [...marqueeLogos, ...marqueeLogos];

  return (
    <section className="py-14">
      <Container>
        <p className="text-center text-sm text-subtle">
          Trusted by modern retailers across every counter
        </p>
      </Container>

      <div className="mask-fade-edges relative mt-8 overflow-hidden">
        <div className="flex w-max animate-marquee items-center gap-14 pr-14">
          {row.map((logo, i) => (
            <span
              key={i}
              className="shrink-0 text-xl font-semibold tracking-tight text-muted/50 transition-colors hover:text-foreground"
            >
              {logo}
            </span>
          ))}
        </div>
      </div>
    </section>
  );
}
