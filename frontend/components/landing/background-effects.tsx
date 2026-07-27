/**
 * Fixed, page-wide ambient backdrop: layered amber aurora blobs, a faint
 * grid, and a vignette. Purely decorative and pointer-transparent.
 */
export function BackgroundEffects() {
  return (
    <div
      aria-hidden
      className="pointer-events-none fixed inset-0 -z-10 overflow-hidden bg-canvas"
    >
      {/* faint grid */}
      <div className="bg-grid mask-fade-b absolute inset-0 opacity-[0.4]" />

      {/* aurora blobs */}
      <div className="absolute -top-40 left-1/2 h-[38rem] w-[38rem] -translate-x-1/2 animate-aurora rounded-full bg-[radial-gradient(circle,rgba(245,158,11,0.22),transparent_62%)] blur-3xl" />
      <div className="absolute top-[20%] -left-40 h-[30rem] w-[30rem] animate-aurora rounded-full bg-[radial-gradient(circle,rgba(251,191,36,0.14),transparent_65%)] blur-3xl [animation-delay:-6s]" />
      <div className="absolute right-[-10%] bottom-[-10%] h-[34rem] w-[34rem] animate-aurora rounded-full bg-[radial-gradient(circle,rgba(180,83,9,0.16),transparent_65%)] blur-3xl [animation-delay:-11s]" />

      {/* top spotlight */}
      <div className="absolute inset-x-0 top-0 h-px bg-linear-to-r from-transparent via-accent/40 to-transparent" />

      {/* vignette */}
      <div className="absolute inset-0 bg-[radial-gradient(120%_80%_at_50%_-10%,transparent_40%,rgba(0,0,0,0.55))]" />
    </div>
  );
}
