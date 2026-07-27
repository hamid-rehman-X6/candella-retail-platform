"use client";

import { ArrowRight } from "lucide-react";
import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { RevealGroup, RevealItem } from "@/components/motion/reveal";
import { industries } from "@/lib/site";

export function Industries() {
  return (
    <section id="industries" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="Industries"
          title="Built for the way"
          highlight="your industry works."
          description="The shared core stays the same. The details — how you track stock, price, and sell — adapt to your trade out of the box."
        />

        <RevealGroup
          className="mt-16 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
          stagger={0.07}
        >
          {industries.map((industry) => {
            const Icon = industry.icon;
            return (
              <RevealItem key={industry.name}>
                <a
                  href="#cta"
                  className="group relative flex items-center gap-4 overflow-hidden rounded-2xl border border-line bg-white/[0.02] p-5 transition-all duration-500 hover:border-accent/30 hover:bg-white/[0.04]"
                >
                  <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-accent/10 text-accent ring-1 ring-accent/20 transition-transform duration-500 group-hover:scale-110">
                    <Icon className="h-6 w-6" />
                  </span>
                  <div className="min-w-0">
                    <h3 className="font-semibold tracking-tight">
                      {industry.name}
                    </h3>
                    <p className="mt-0.5 truncate text-sm text-muted">
                      {industry.blurb}
                    </p>
                  </div>
                  <ArrowRight className="ml-auto h-4 w-4 shrink-0 -translate-x-2 text-subtle opacity-0 transition-all duration-500 group-hover:translate-x-0 group-hover:text-accent group-hover:opacity-100" />
                </a>
              </RevealItem>
            );
          })}
        </RevealGroup>
      </Container>
    </section>
  );
}
