"use client";

import { useRef } from "react";
import {
  motion,
  useScroll,
  useTransform,
  useReducedMotion,
} from "motion/react";
import { ArrowRight, PlayCircle, Wifi, Zap } from "lucide-react";
import { Container } from "@/components/ui/container";
import { Button } from "@/components/ui/button";
import { DashboardMockup } from "@/components/landing/dashboard-mockup";
import { trustFeatures } from "@/lib/site";

const headline = ["One", "platform,", "every", "retail", "business."];

export function Hero() {
  const ref = useRef<HTMLDivElement>(null);
  const reduce = useReducedMotion();
  const { scrollYProgress } = useScroll({
    target: ref,
    offset: ["start start", "end start"],
  });

  const mockY = useTransform(scrollYProgress, [0, 1], [0, reduce ? 0 : 140]);
  const mockRotate = useTransform(
    scrollYProgress,
    [0, 1],
    [0, reduce ? 0 : -4],
  );
  const mockScale = useTransform(
    scrollYProgress,
    [0, 1],
    [1, reduce ? 1 : 0.92],
  );
  const mockOpacity = useTransform(
    scrollYProgress,
    [0, 0.8],
    [1, reduce ? 1 : 0.4],
  );

  return (
    <section
      id="top"
      ref={ref}
      className="relative overflow-hidden pt-36 sm:pt-44"
    >
      <Container>
        <div className="flex flex-col items-center text-center">
          {/* announcement pill */}
          <motion.a
            href="#products"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="group mb-8 inline-flex items-center gap-2 rounded-full border border-line bg-white/5 py-1.5 pr-4 pl-2 text-sm text-muted backdrop-blur-sm transition-colors hover:border-line-strong"
          >
            <span className="rounded-full bg-accent/15 px-2.5 py-0.5 text-xs font-medium text-accent">
              New
            </span>
            The complete retail operating system
            <ArrowRight className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5" />
          </motion.a>

          {/* headline — word by word */}
          <h1 className="max-w-4xl text-5xl leading-[1.02] font-semibold tracking-tight text-balance sm:text-6xl md:text-7xl">
            {headline.map((word, i) => (
              <motion.span
                key={i}
                initial={{ opacity: 0, y: 28, filter: "blur(8px)" }}
                animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                transition={{
                  duration: 0.7,
                  delay: 0.15 + i * 0.09,
                  ease: [0.22, 1, 0.36, 1],
                }}
                className="mr-[0.25em] inline-block"
              >
                {word === "every" ||
                word === "retail" ||
                word === "business." ? (
                  <span className="text-gradient">{word}</span>
                ) : (
                  word
                )}
              </motion.span>
            ))}
          </h1>

          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, delay: 0.7 }}
            className="mt-7 max-w-2xl text-lg leading-relaxed text-muted sm:text-xl"
          >
            POS, inventory, pharmacy, garments, cosmetics, CRM and analytics —
            unified in one platform that runs your entire business, across every
            store and every counter.
          </motion.p>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, delay: 0.82 }}
            className="mt-9 flex flex-col items-center gap-3 sm:flex-row"
          >
            <Button href="/signup" size="lg">
              Start free trial
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Button>
            <Button href="#ecosystem" variant="secondary" size="lg">
              <PlayCircle className="h-4 w-4" />
              See how it works
            </Button>
          </motion.div>

          {/* trust row */}
          <motion.ul
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.7, delay: 1 }}
            className="mt-10 flex flex-wrap items-center justify-center gap-x-6 gap-y-3 text-sm text-subtle"
          >
            {trustFeatures.map((f) => {
              const Icon = f.icon;
              return (
                <li key={f.label} className="inline-flex items-center gap-2">
                  <Icon className="h-4 w-4 text-accent" />
                  {f.label}
                </li>
              );
            })}
          </motion.ul>
        </div>

        {/* dashboard preview with parallax */}
        <motion.div
          style={{
            y: mockY,
            rotateX: mockRotate,
            scale: mockScale,
            opacity: mockOpacity,
          }}
          initial={{ opacity: 0, y: 60 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 1, delay: 0.5, ease: [0.22, 1, 0.36, 1] }}
          className="relative mx-auto mt-16 max-w-5xl perspective-[2000px] sm:mt-20"
        >
          {/* glow behind */}
          <div className="absolute -inset-x-10 -top-10 bottom-0 -z-10 bg-[radial-gradient(60%_60%_at_50%_0%,rgba(245,158,11,0.28),transparent_70%)] blur-2xl" />

          <DashboardMockup />

          {/* floating chips */}
          <motion.div
            animate={reduce ? undefined : { y: [0, -14, 0] }}
            transition={{ duration: 7, repeat: Infinity, ease: "easeInOut" }}
            className="glass absolute top-24 -left-4 hidden items-center gap-2 rounded-2xl px-4 py-3 shadow-xl sm:flex"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-400/15 text-emerald-400">
              <Wifi className="h-4 w-4" />
            </span>
            <div className="text-left">
              <p className="text-xs font-medium">Offline sale synced</p>
              <p className="text-[10px] text-subtle">2 seconds ago</p>
            </div>
          </motion.div>

          <motion.div
            animate={reduce ? undefined : { y: [0, 16, 0] }}
            transition={{ duration: 9, repeat: Infinity, ease: "easeInOut" }}
            className="glass absolute -right-4 bottom-28 hidden items-center gap-2 rounded-2xl px-4 py-3 shadow-xl sm:flex"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-accent/15 text-accent">
              <Zap className="h-4 w-4" />
            </span>
            <div className="text-left">
              <p className="text-xs font-medium">Reorder triggered</p>
              <p className="text-[10px] text-subtle">Ibuprofen · 3 stores</p>
            </div>
          </motion.div>
        </motion.div>
      </Container>
    </section>
  );
}
