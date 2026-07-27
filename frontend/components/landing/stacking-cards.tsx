"use client";

import { useRef } from "react";
import {
  motion,
  useScroll,
  useTransform,
  useReducedMotion,
  type MotionValue,
} from "motion/react";
import { Check } from "lucide-react";
import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { flowCards, type FlowCard } from "@/lib/site";

function StackCard({
  card,
  index,
  total,
  progress,
}: {
  card: FlowCard;
  index: number;
  total: number;
  progress: MotionValue<number>;
}) {
  const Icon = card.icon;
  // Earlier cards shrink slightly as later ones stack over them.
  const targetScale = 1 - (total - 1 - index) * 0.045;
  const scale = useTransform(progress, [index / total, 1], [1, targetScale]);

  return (
    <div className="sticky top-0 flex h-screen items-center justify-center">
      <motion.div
        style={{ scale, top: `calc(-6vh + ${index * 26}px)` }}
        className="card-surface ring-gradient glow-accent relative w-full max-w-4xl overflow-hidden rounded-3xl"
      >
        <div className="grid gap-8 p-8 sm:p-12 md:grid-cols-[1.1fr_0.9fr]">
          <div className="flex flex-col">
            <span className="text-sm font-semibold text-accent">
              Step {card.step}
            </span>
            <h3 className="mt-3 text-3xl font-semibold tracking-tight sm:text-4xl">
              {card.title}
            </h3>
            <p className="mt-4 text-base leading-relaxed text-muted">
              {card.description}
            </p>
            <ul className="mt-6 flex flex-wrap gap-2">
              {card.points.map((p) => (
                <li
                  key={p}
                  className="inline-flex items-center gap-1.5 rounded-full border border-line bg-white/5 px-3 py-1.5 text-xs text-foreground/85"
                >
                  <Check className="h-3 w-3 text-accent" />
                  {p}
                </li>
              ))}
            </ul>
          </div>

          {/* visual panel */}
          <div className="relative flex items-center justify-center overflow-hidden rounded-2xl border border-line bg-canvas-raised/60 p-8">
            <div className="bg-dots absolute inset-0 opacity-40" />
            <div className="absolute -top-10 -right-10 h-32 w-32 rounded-full bg-[radial-gradient(circle,rgba(245,158,11,0.35),transparent_70%)] blur-2xl" />
            <div className="relative flex flex-col items-center gap-4">
              <span className="flex h-20 w-20 items-center justify-center rounded-3xl bg-linear-to-br from-accent-bright to-accent-deep text-black shadow-[0_0_50px_-8px_rgba(245,158,11,0.7)]">
                <Icon className="h-9 w-9" />
              </span>
              <span className="text-7xl font-bold text-white/5">
                {card.step}
              </span>
            </div>
          </div>
        </div>
      </motion.div>
    </div>
  );
}

export function StackingCards() {
  const container = useRef<HTMLDivElement>(null);
  const reduce = useReducedMotion();
  const { scrollYProgress } = useScroll({
    target: container,
    offset: ["start start", "end end"],
  });

  return (
    <section id="how" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="How it flows"
          title="One sale sets your"
          highlight="whole business in motion."
          description="Watch what happens the instant a customer checks out — every part of Candella reacts in real time, with zero manual work."
        />
      </Container>

      {reduce ? (
        // Reduced-motion: simple stacked layout, no scroll animation.
        <Container className="mt-14 flex flex-col gap-6">
          {flowCards.map((card) => (
            <div
              key={card.step}
              className="card-surface ring-gradient relative overflow-hidden rounded-3xl p-8"
            >
              <span className="text-sm font-semibold text-accent">
                Step {card.step}
              </span>
              <h3 className="mt-2 text-2xl font-semibold">{card.title}</h3>
              <p className="mt-3 text-muted">{card.description}</p>
            </div>
          ))}
        </Container>
      ) : (
        <div ref={container} className="relative mt-14">
          <Container>
            {flowCards.map((card, i) => (
              <StackCard
                key={card.step}
                card={card}
                index={i}
                total={flowCards.length}
                progress={scrollYProgress}
              />
            ))}
          </Container>
        </div>
      )}
    </section>
  );
}
