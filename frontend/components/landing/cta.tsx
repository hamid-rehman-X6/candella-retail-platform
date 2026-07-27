"use client";

import { ArrowRight } from "lucide-react";
import { Container } from "@/components/ui/container";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/motion/reveal";

export function CTA() {
  return (
    <section id="cta" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <Reveal>
          <div className="relative overflow-hidden rounded-[2rem] border border-line px-6 py-16 text-center sm:px-16 sm:py-24">
            {/* glow field */}
            <div className="absolute inset-0 -z-10 bg-[radial-gradient(80%_140%_at_50%_0%,rgba(245,158,11,0.22),transparent_60%)]" />
            <div className="bg-dots absolute inset-0 -z-10 opacity-30" />
            <div className="absolute inset-x-0 top-0 h-px bg-linear-to-r from-transparent via-accent/60 to-transparent" />

            <h2 className="mx-auto max-w-3xl text-4xl leading-[1.05] font-semibold tracking-tight text-balance sm:text-5xl md:text-6xl">
              Run your whole retail business on{" "}
              <span className="text-gradient">one platform.</span>
            </h2>
            <p className="mx-auto mt-6 max-w-xl text-lg text-muted">
              Join the retailers who replaced their patchwork of tools with
              Candella. Set up in minutes, live the same day.
            </p>

            <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Button href="/signup" size="lg">
                Start free trial
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </Button>
              <Button href="/login" variant="secondary" size="lg">
                Sign in
              </Button>
            </div>

            <p className="mt-6 text-sm text-subtle">
              14-day free trial · No credit card required · Cancel anytime
            </p>
          </div>
        </Reveal>
      </Container>
    </section>
  );
}
