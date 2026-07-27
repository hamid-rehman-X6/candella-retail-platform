"use client";

import { useState } from "react";
import { motion, AnimatePresence } from "motion/react";
import { Plus } from "lucide-react";
import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { Reveal } from "@/components/motion/reveal";
import { faqs } from "@/lib/site";
import { cn } from "@/lib/cn";

function FaqItem({ q, a }: { q: string; a: string }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="rounded-2xl border border-line bg-white/[0.02] transition-colors hover:border-line-strong">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-4 px-6 py-5 text-left"
        aria-expanded={open}
      >
        <span className="font-medium tracking-tight">{q}</span>
        <span
          className={cn(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-line text-muted transition-all duration-300",
            open && "rotate-45 border-accent/40 bg-accent/10 text-accent",
          )}
        >
          <Plus className="h-4 w-4" />
        </span>
      </button>
      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden"
          >
            <p className="px-6 pb-5 text-sm leading-relaxed text-muted">{a}</p>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

export function Faq() {
  return (
    <section id="faq" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="FAQ"
          title="Questions,"
          highlight="answered."
          description="Everything you need to know before getting started. Still curious? Our team is one message away."
        />

        <div className="mx-auto mt-14 flex max-w-3xl flex-col gap-3">
          {faqs.map((item, i) => (
            <Reveal key={item.q} delay={i * 0.04}>
              <FaqItem q={item.q} a={item.a} />
            </Reveal>
          ))}
        </div>
      </Container>
    </section>
  );
}
