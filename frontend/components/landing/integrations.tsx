import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { Marquee } from "@/components/ui/marquee";
import { integrations } from "@/lib/site";

function Pill({ name }: { name: string }) {
  return (
    <span className="inline-flex shrink-0 items-center gap-2 rounded-2xl border border-line bg-white/[0.03] px-5 py-3 text-sm font-medium text-muted">
      <span className="h-2 w-2 rounded-full bg-accent/70" />
      {name}
    </span>
  );
}

export function Integrations() {
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

      {/* Two rows scrolling in opposite directions. Each row carries the full
          integration list, so the row stays dense at any width. */}
      <div className="mt-14 flex flex-col gap-4">
        <Marquee gap="1rem" durationSeconds={46}>
          {integrations.map((name) => (
            <Pill key={name} name={name} />
          ))}
        </Marquee>
        <Marquee gap="1rem" durationSeconds={46} reverse>
          {integrations.map((name) => (
            <Pill key={name} name={name} />
          ))}
        </Marquee>
      </div>
    </section>
  );
}
