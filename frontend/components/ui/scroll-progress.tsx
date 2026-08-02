"use client";

import { motion, useScroll, useSpring } from "motion/react";

/**
 * A thin amber bar pinned to the very top of the viewport that fills left→right
 * as the page scrolls. Driven by the document scroll progress (0→1) and smoothed
 * with a spring so it glides rather than jumps.
 */
export function ScrollProgress() {
  const { scrollYProgress } = useScroll();
  const scaleX = useSpring(scrollYProgress, {
    stiffness: 120,
    damping: 30,
    mass: 0.3,
  });

  return (
    <motion.div
      aria-hidden
      style={{ scaleX }}
      className="fixed inset-x-0 top-0 z-[60] h-0.5 origin-left bg-linear-to-r from-accent-deep via-accent to-accent-bright"
    />
  );
}
