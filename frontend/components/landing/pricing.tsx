"use client";

import { Check, ArrowRight } from "lucide-react";
import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { Button } from "@/components/ui/button";
import { RevealGroup, RevealItem } from "@/components/motion/reveal";
import { pricingTiers } from "@/lib/site";
import { cn } from "@/lib/cn";

export function Pricing() {
  return (
    <section id="pricing" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="Pricing"
          title="Simple pricing that"
          highlight="scales with you."
          description="Start free for 14 days. No credit card required. Change plans anytime as your business grows."
        />

        <RevealGroup
          className="mt-16 grid items-stretch gap-6 lg:grid-cols-3"
          stagger={0.09}
        >
          {pricingTiers.map((tier) => (
            <RevealItem key={tier.name} className="h-full">
              <div
                className={cn(
                  "relative flex h-full flex-col overflow-hidden rounded-3xl p-8 transition-all duration-500",
                  tier.featured
                    ? "card-surface ring-gradient glow-accent scale-[1.02]"
                    : "border border-line bg-white/[0.02] hover:border-line-strong",
                )}
              >
                {tier.featured && (
                  <span className="absolute top-6 right-6 rounded-full bg-accent/15 px-3 py-1 text-xs font-medium text-accent">
                    Most popular
                  </span>
                )}

                <h3 className="text-lg font-semibold tracking-tight">
                  {tier.name}
                </h3>
                <p className="mt-2 text-sm text-muted">{tier.description}</p>

                <div className="mt-6 flex items-end gap-1.5">
                  <span className="text-5xl font-semibold tracking-tight">
                    {tier.price}
                  </span>
                  <span className="mb-1.5 text-sm text-subtle">
                    {tier.period}
                  </span>
                </div>

                <Button
                  href="#cta"
                  variant={tier.featured ? "primary" : "secondary"}
                  className="mt-7 w-full"
                >
                  {tier.cta}
                  <ArrowRight className="h-4 w-4" />
                </Button>

                <ul className="mt-8 space-y-3 border-t border-line pt-7">
                  {tier.features.map((f) => (
                    <li key={f} className="flex items-start gap-3 text-sm">
                      <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-accent/15 text-accent">
                        <Check className="h-3 w-3" />
                      </span>
                      <span className="text-foreground/85">{f}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </RevealItem>
          ))}
        </RevealGroup>
      </Container>
    </section>
  );
}
