import type { LucideIcon } from "lucide-react";

export function AuthHeader({
  icon: Icon,
  title,
  subtitle,
}: {
  icon?: LucideIcon;
  title: string;
  subtitle?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col">
      {Icon && (
        <span className="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-accent/10 text-accent ring-1 ring-accent/20">
          <Icon className="h-6 w-6" />
        </span>
      )}
      <h1 className="text-2xl font-semibold tracking-tight sm:text-[1.75rem]">
        {title}
      </h1>
      {subtitle && (
        <p className="mt-2 text-sm leading-relaxed text-muted">{subtitle}</p>
      )}
    </div>
  );
}
