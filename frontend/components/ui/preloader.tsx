"use client";

import { useEffect, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { LogoMark } from "@/components/ui/logo";

const MIN_DURATION = 700; // never flash for less than this
const HARD_CAP = 2500; // never block longer than this

/**
 * Full-screen branded loading overlay shown on initial page load. It sits
 * above everything, fills a fake progress toward ~90%, completes to 100% once
 * the window has loaded (or the hard cap hits), then fades out and unmounts.
 * Client navigations don't replay it because the root layout stays mounted.
 */
export function Preloader() {
  const reduce = useReducedMotion();
  const [visible, setVisible] = useState(true);
  const [progress, setProgress] = useState(8);

  // Drive progress + decide when to finish.
  useEffect(() => {
    const start = performance.now();
    let finished = false;

    const interval = setInterval(() => {
      setProgress((p) => (p >= 90 ? p : p + Math.max(0.6, (90 - p) * 0.06)));
    }, 40);

    const finish = () => {
      if (finished) return;
      finished = true;
      const wait = Math.max(0, MIN_DURATION - (performance.now() - start));
      window.setTimeout(() => {
        clearInterval(interval);
        setProgress(100);
        window.setTimeout(() => setVisible(false), reduce ? 0 : 400);
      }, wait);
    };

    if (document.readyState === "complete") finish();
    else window.addEventListener("load", finish, { once: true });

    const cap = window.setTimeout(finish, HARD_CAP);

    return () => {
      clearInterval(interval);
      clearTimeout(cap);
      window.removeEventListener("load", finish);
    };
  }, [reduce]);

  // Lock scroll while the overlay is up.
  useEffect(() => {
    document.body.style.overflow = visible ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [visible]);

  return (
    <AnimatePresence>
      {visible && (
        <motion.div
          key="preloader"
          aria-hidden
          initial={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
          className="fixed inset-0 z-[100] flex flex-col items-center justify-center bg-canvas"
        >
          {/* ambient center glow */}
          <div className="pointer-events-none absolute h-[28rem] w-[28rem] rounded-full bg-[radial-gradient(circle,rgba(245,158,11,0.16),transparent_65%)] blur-3xl" />

          <div className="relative flex flex-col items-center">
            {/* pulsing halo behind the mark */}
            <div className="relative flex items-center justify-center">
              <motion.span
                aria-hidden
                className="absolute h-24 w-24 rounded-3xl bg-accent/25 blur-2xl"
                animate={
                  reduce
                    ? undefined
                    : { scale: [1, 1.25, 1], opacity: [0.5, 0.9, 0.5] }
                }
                transition={{
                  duration: 2.2,
                  repeat: Infinity,
                  ease: "easeInOut",
                }}
              />
              <motion.div
                initial={{ scale: 0.8, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
              >
                <motion.div
                  animate={reduce ? undefined : { y: [0, -6, 0] }}
                  transition={{
                    duration: 2.4,
                    repeat: Infinity,
                    ease: "easeInOut",
                  }}
                >
                  <LogoMark className="h-16 w-16 rounded-2xl" />
                </motion.div>
              </motion.div>
            </div>

            <motion.p
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.25, duration: 0.5 }}
              className="mt-6 text-lg font-semibold tracking-tight text-foreground"
            >
              Candella
            </motion.p>

            {/* progress bar */}
            <div className="mt-6 h-1 w-44 overflow-hidden rounded-full bg-white/10">
              <div
                className="h-full rounded-full bg-linear-to-r from-accent-deep via-accent to-accent-bright transition-[width] duration-300 ease-out"
                style={{ width: `${progress}%` }}
              />
            </div>

            <motion.p
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.4, duration: 0.5 }}
              className="mt-4 text-xs tracking-wide text-subtle"
            >
              Preparing your workspace…
            </motion.p>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
