"use client";

import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { RevealGroup, RevealItem } from "@/components/motion/reveal";
import { features } from "@/lib/site";
import { cn } from "@/lib/cn";

export function Features() {
  return (
    <section id="features" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="Platform capabilities"
          title="Everything a serious retailer"
          highlight="expects — and more."
          description="The foundations that keep hundreds of stores running smoothly, day in and day out."
        />

        <RevealGroup
          className="mt-16 grid auto-rows-[minmax(180px,auto)] grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4"
          stagger={0.06}
        >
          {features.map((feature) => {
            const Icon = feature.icon;
            return (
              <RevealItem
                key={feature.title}
                className={cn(feature.span === "wide" && "sm:col-span-2")}
              >
                <div className="group relative flex h-full flex-col overflow-hidden rounded-3xl border border-line bg-white/[0.02] p-6 transition-all duration-500 hover:border-line-strong hover:bg-white/[0.04]">
                  <div className="pointer-events-none absolute -top-12 -right-12 h-32 w-32 rounded-full bg-[radial-gradient(circle,rgba(245,158,11,0.14),transparent_70%)] opacity-0 blur-2xl transition-opacity duration-500 group-hover:opacity-100" />
                  <span className="flex h-11 w-11 items-center justify-center rounded-2xl bg-accent/10 text-accent ring-1 ring-accent/20">
                    <Icon className="h-5 w-5" />
                  </span>
                  <h3 className="mt-5 text-lg font-semibold tracking-tight">
                    {feature.title}
                  </h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted">
                    {feature.description}
                  </p>
                </div>
              </RevealItem>
            );
          })}
        </RevealGroup>
      </Container>
    </section>
  );
}
