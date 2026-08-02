"use client";

import { useState } from "react";
import {
  AnimatePresence,
  motion,
  useScroll,
  useSpring,
  useMotionValueEvent,
  useReducedMotion,
} from "motion/react";
import { ArrowUp } from "lucide-react";

const SHOW_AFTER = 500; // px scrolled before the button appears

/**
 * A floating, bottom-right button that fades in once the user has scrolled past
 * SHOW_AFTER, shows the page's scroll progress as a ring around an up-arrow, and
 * smooth-scrolls back to the top on click. Honors reduced motion and is
 * keyboard / screen-reader accessible.
 */
export function ScrollToTop() {
  const reduce = useReducedMotion();
  const { scrollY, scrollYProgress } = useScroll();
  // Smoothed 0→1 progress that drives the ring (pathLength).
  const progress = useSpring(scrollYProgress, {
    stiffness: 120,
    damping: 30,
    mass: 0.3,
  });
  const [visible, setVisible] = useState(false);

  useMotionValueEvent(scrollY, "change", (y) => {
    setVisible(y > SHOW_AFTER);
  });

  function toTop() {
    window.scrollTo({ top: 0, behavior: reduce ? "auto" : "smooth" });
  }

  return (
    <AnimatePresence>
      {visible && (
        <motion.button
          type="button"
          onClick={toTop}
          aria-label="Scroll back to top"
          initial={{ opacity: 0, scale: 0.8, y: 12 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.8, y: 12 }}
          transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
          className="glass group fixed right-6 bottom-6 z-50 flex h-12 w-12 items-center justify-center rounded-full text-foreground shadow-[0_8px_30px_-8px_rgba(0,0,0,0.6)] transition-colors hover:border-line-strong focus-visible:ring-2 focus-visible:ring-accent/60 focus-visible:outline-none"
        >
          {/* Scroll-progress ring. Rotated -90° so it starts filling from the top. */}
          <svg
            viewBox="0 0 48 48"
            className="absolute inset-0 -rotate-90"
            aria-hidden
          >
            <circle
              cx="24"
              cy="24"
              r="22"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              className="text-white/10"
            />
            <motion.circle
              cx="24"
              cy="24"
              r="22"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              className="text-accent"
              style={{ pathLength: progress }}
            />
          </svg>

          <ArrowUp className="relative h-5 w-5 transition-transform group-hover:-translate-y-0.5 group-hover:text-accent" />
        </motion.button>
      )}
    </AnimatePresence>
  );
}
