import { Container } from "@/components/ui/container";
import { AnimatedCounter } from "@/components/motion/animated-counter";
import { Reveal } from "@/components/motion/reveal";
import { stats } from "@/lib/site";

export function Stats() {
  return (
    <section className="relative py-20">
      <Container>
        <Reveal>
          <div className="card-surface relative overflow-hidden rounded-3xl border border-line">
            <div className="absolute inset-0 bg-[radial-gradient(80%_120%_at_50%_0%,rgba(245,158,11,0.12),transparent_60%)]" />
            <div className="relative grid gap-x-6 gap-y-10 p-10 sm:grid-cols-2 sm:p-14 lg:grid-cols-4">
              {stats.map((stat) => (
                <div key={stat.label} className="text-center">
                  <div className="text-gradient text-5xl font-semibold tracking-tight sm:text-6xl">
                    <AnimatedCounter
                      value={stat.value}
                      decimals={stat.decimals ?? 0}
                      prefix={stat.prefix ?? ""}
                      suffix={stat.suffix ?? ""}
                    />
                  </div>
                  <p className="mx-auto mt-3 max-w-[12rem] text-sm text-muted">
                    {stat.label}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </Reveal>
      </Container>
    </section>
  );
}
