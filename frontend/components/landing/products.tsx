"use client";

import { useRef } from "react";
import { useReducedMotion } from "motion/react";
import { Check, ArrowUpRight } from "lucide-react";
import { Container } from "@/components/ui/container";
import { SectionHeading } from "@/components/ui/section-heading";
import { RevealGroup, RevealItem } from "@/components/motion/reveal";
import { productModules, type ProductModule } from "@/lib/site";
import { cn } from "@/lib/cn";

function ProductCard({ product }: { product: ProductModule }) {
  const cardRef = useRef<HTMLDivElement>(null);
  const reduce = useReducedMotion();
  const Icon = product.icon;

  // pointer-tracked spotlight
  function onMove(e: React.MouseEvent<HTMLDivElement>) {
    if (reduce || !cardRef.current) return;
    const rect = cardRef.current.getBoundingClientRect();
    cardRef.current.style.setProperty("--mx", `${e.clientX - rect.left}px`);
    cardRef.current.style.setProperty("--my", `${e.clientY - rect.top}px`);
  }

  return (
    <div
      ref={cardRef}
      onMouseMove={onMove}
      className="group relative flex h-full flex-col overflow-hidden rounded-3xl border border-line bg-white/[0.02] p-7 transition-all duration-500 hover:border-line-strong hover:bg-white/[0.04]"
    >
      {/* spotlight follows cursor */}
      <div
        className="pointer-events-none absolute inset-0 opacity-0 transition-opacity duration-500 group-hover:opacity-100"
        style={{
          background:
            "radial-gradient(340px circle at var(--mx,50%) var(--my,0%), rgba(245,158,11,0.12), transparent 70%)",
        }}
      />
      {/* accent wash */}
      <div
        className={cn(
          "pointer-events-none absolute -top-16 -right-16 h-40 w-40 rounded-full bg-linear-to-br opacity-50 blur-2xl transition-opacity duration-500 group-hover:opacity-90",
          product.accent,
        )}
      />

      <div className="relative flex items-center justify-between">
        <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-accent/10 text-accent ring-1 ring-accent/20 transition-transform duration-500 group-hover:scale-110">
          <Icon className="h-6 w-6" />
        </span>
        <ArrowUpRight className="h-5 w-5 text-subtle transition-all duration-500 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 group-hover:text-accent" />
      </div>

      <div className="relative mt-6">
        <p className="text-xs font-medium tracking-widest text-accent/80 uppercase">
          {product.tagline}
        </p>
        <h3 className="mt-2 text-xl font-semibold tracking-tight">
          {product.name}
        </h3>
        <p className="mt-2.5 text-sm leading-relaxed text-muted">
          {product.description}
        </p>
      </div>

      <ul className="relative mt-6 space-y-2.5 border-t border-line pt-5">
        {product.features.map((f) => (
          <li
            key={f}
            className="flex items-center gap-2.5 text-sm text-foreground/85"
          >
            <span className="flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-accent/15 text-accent">
              <Check className="h-3 w-3" />
            </span>
            {f}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function Products() {
  return (
    <section id="products" className="relative scroll-mt-24 py-24 sm:py-32">
      <Container>
        <SectionHeading
          eyebrow="Products"
          title="Six powerful modules,"
          highlight="one seamless platform."
          description="Turn on what you need today, add the rest as you grow. Every module shares the same products, stock, customers and payments — so nothing ever falls out of sync."
        />

        <RevealGroup
          className="mt-16 grid gap-5 sm:grid-cols-2 lg:grid-cols-3"
          stagger={0.08}
        >
          {productModules.map((product) => (
            <RevealItem key={product.id} className="h-full">
              <ProductCard product={product} />
            </RevealItem>
          ))}
        </RevealGroup>
      </Container>
    </section>
  );
}
