"use client";

import { motion, useReducedMotion } from "motion/react";
import {
  Store,
  Boxes,
  Users,
  BarChart3,
  CreditCard,
  Layers,
} from "lucide-react";
import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { Reveal, RevealGroup, RevealItem } from "@/components/motion/reveal";
import { ecosystemPillars } from "@/lib/site";

const orbit = [
  { icon: Store, label: "POS" },
  { icon: Boxes, label: "Inventory" },
  { icon: Users, label: "CRM" },
  { icon: CreditCard, label: "Payments" },
  { icon: BarChart3, label: "Analytics" },
];

export function Ecosystem() {
  const reduce = useReducedMotion();

  return (
    <section id="ecosystem" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="The ecosystem"
          title="Not another tool. Your entire"
          highlight="retail operating system."
          description="Most retailers stitch together a POS here, a spreadsheet there, a separate loyalty app. Candella replaces the patchwork with one connected core — so every part of your business speaks the same language, in real time."
        />

        <div className="mt-16 grid items-center gap-12 lg:grid-cols-2">
          {/* orbit visual */}
          <Reveal className="order-2 lg:order-1">
            <div className="relative mx-auto flex aspect-square w-full max-w-md items-center justify-center">
              {/* rings */}
              <div className="absolute inset-0 rounded-full border border-line" />
              <div className="absolute inset-[14%] rounded-full border border-line" />
              <div className="absolute inset-[28%] rounded-full border border-line/70" />

              {/* rotating orbit of module chips */}
              <motion.div
                className="absolute inset-0"
                animate={reduce ? undefined : { rotate: 360 }}
                transition={{ duration: 40, repeat: Infinity, ease: "linear" }}
              >
                {orbit.map((node, i) => {
                  const angle = (i / orbit.length) * Math.PI * 2;
                  const radius = 46; // % from center
                  const x = 50 + radius * Math.cos(angle);
                  const y = 50 + radius * Math.sin(angle);
                  const Icon = node.icon;
                  return (
                    <motion.div
                      key={node.label}
                      className="card-surface absolute flex h-14 w-14 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-2xl"
                      style={{ left: `${x}%`, top: `${y}%` }}
                      animate={reduce ? undefined : { rotate: -360 }}
                      transition={{
                        duration: 40,
                        repeat: Infinity,
                        ease: "linear",
                      }}
                    >
                      <Icon className="h-5 w-5 text-accent" />
                    </motion.div>
                  );
                })}
              </motion.div>

              {/* core */}
              <div className="relative z-10 flex h-28 w-28 flex-col items-center justify-center rounded-3xl bg-linear-to-br from-accent-bright to-accent-deep text-black shadow-[0_0_60px_-10px_rgba(245,158,11,0.7)]">
                <Layers className="h-7 w-7" />
                <span className="mt-1 text-xs font-semibold">Shared core</span>
              </div>
            </div>
          </Reveal>

          {/* pillars */}
          <RevealGroup className="order-1 flex flex-col gap-4 lg:order-2">
            {ecosystemPillars.map((pillar) => {
              const Icon = pillar.icon;
              return (
                <RevealItem key={pillar.title}>
                  <div className="group flex gap-4 rounded-2xl border border-line bg-white/[0.02] p-5 transition-colors hover:border-line-strong hover:bg-white/[0.04]">
                    <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-accent/10 text-accent ring-1 ring-accent/20">
                      <Icon className="h-5 w-5" />
                    </span>
                    <div>
                      <h3 className="font-semibold tracking-tight">
                        {pillar.title}
                      </h3>
                      <p className="mt-1 text-sm leading-relaxed text-muted">
                        {pillar.description}
                      </p>
                    </div>
                  </div>
                </RevealItem>
              );
            })}
          </RevealGroup>
        </div>
      </Container>
    </section>
  );
}
