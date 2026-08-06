import { Container } from "@/components/ui/container";
import { Marquee } from "@/components/ui/marquee";
import { marqueeLogos } from "@/lib/site";

/** Infinite auto-scrolling row of customer wordmarks. */
export function LogoMarquee() {
  return (
    <section className="py-14">
      <Container>
        <p className="text-center text-sm text-subtle">
          Trusted by modern retailers across every counter
        </p>
      </Container>

      <div className="mt-8">
        <Marquee gap="3.5rem" durationSeconds={42}>
          {marqueeLogos.map((logo) => (
            <span
              key={logo}
              className="shrink-0 text-xl font-semibold tracking-tight text-muted/50 transition-colors hover:text-foreground"
            >
              {logo}
            </span>
          ))}
        </Marquee>
      </div>
    </section>
  );
}
