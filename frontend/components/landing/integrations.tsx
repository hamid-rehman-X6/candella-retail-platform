import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { integrations } from "@/lib/site";

function Row({ items, reverse }: { items: string[]; reverse?: boolean }) {
  const doubled = [...items, ...items];
  return (
    <div
      className={`flex w-max gap-4 pr-4 ${
        reverse ? "animate-marquee-reverse" : "animate-marquee"
      }`}
    >
      {doubled.map((name, i) => (
        <span
          key={`${name}-${i}`}
          className="inline-flex shrink-0 items-center gap-2 rounded-2xl border border-line bg-white/[0.03] px-5 py-3 text-sm font-medium text-muted"
        >
          <span className="h-2 w-2 rounded-full bg-accent/70" />
          {name}
        </span>
      ))}
    </div>
  );
}

export function Integrations() {
  const half = Math.ceil(integrations.length / 2);
  return (
    <section className="relative py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="Integrations"
          title="Connects with the tools"
          highlight="you already run on."
          description="Payments, accounting, messaging and commerce — Candella plugs into your stack so data flows without copy-paste."
        />
      </Container>

      <div className="mask-fade-edges mt-14 flex flex-col gap-4 overflow-hidden">
        <Row items={integrations.slice(0, half)} />
        <Row items={integrations.slice(half)} reverse />
      </div>
    </section>
  );
}
