"use client";

import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/cn";

export type SelectOption = { value: string; label: string; disabled?: boolean };

type SelectProps = {
  ariaLabel: string;
  value: string;
  options: SelectOption[];
  onValueChange: (value: string) => void;
  className?: string;
  disabled?: boolean;
  name?: string;
  tone?: "amber" | "rose";
};

export function Select({
  ariaLabel,
  value,
  options,
  onValueChange,
  className,
  disabled = false,
  name,
  tone = "amber",
}: SelectProps) {
  const id = useId();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const [position, setPosition] = useState<React.CSSProperties>({});
  const selected = options.find((option) => option.value === value);
  const isOpen = open && !disabled && options.length > 0;
  const highlight = tone === "rose" ? "text-rose-300" : "text-accent";

  function firstEnabled() {
    return options.findIndex((option) => !option.disabled);
  }

  function move(from: number, direction: number) {
    for (let step = 1; step <= options.length; step++) {
      const index =
        (from + direction * step + options.length * 2) % options.length;
      if (!options[index]?.disabled) return index;
    }
    return from;
  }

  function choose(index: number) {
    const option = options[index];
    if (!option || option.disabled) return;
    onValueChange(option.value);
    setOpen(false);
    triggerRef.current?.focus();
  }

  function openMenu() {
    if (disabled || options.length === 0) return;
    const selectedIndex = options.findIndex(
      (option) => option.value === value && !option.disabled,
    );
    const initial = selectedIndex >= 0 ? selectedIndex : firstEnabled();
    if (initial < 0) return;
    setActive(initial);
    setOpen(true);
  }

  useLayoutEffect(() => {
    if (!isOpen || !triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    const height = Math.min(280, options.length * 40 + 12);
    const spaceBelow = window.innerHeight - rect.bottom - 14;
    const spaceAbove = rect.top - 14;
    const above = spaceBelow < height && spaceAbove > spaceBelow;
    const width = Math.min(rect.width, window.innerWidth - 16);
    const available = above ? spaceAbove : spaceBelow;
    const visibleHeight = Math.min(height, Math.max(48, available));
    setPosition({
      position: "fixed",
      left: Math.max(8, Math.min(rect.left, window.innerWidth - width - 8)),
      top: above ? rect.top - visibleHeight - 6 : rect.bottom + 6,
      width,
      maxHeight: visibleHeight,
    });
  }, [isOpen, options.length]);

  useEffect(() => {
    if (!isOpen) return;
    function closeOutside(event: PointerEvent) {
      const target = event.target as Node;
      if (
        !triggerRef.current?.contains(target) &&
        !menuRef.current?.contains(target)
      )
        setOpen(false);
    }
    function closeOnScroll(event: Event) {
      if (
        event.target instanceof Node &&
        menuRef.current?.contains(event.target)
      )
        return;
      setOpen(false);
    }
    document.addEventListener("pointerdown", closeOutside);
    window.addEventListener("resize", closeOnScroll);
    window.addEventListener("scroll", closeOnScroll, true);
    return () => {
      document.removeEventListener("pointerdown", closeOutside);
      window.removeEventListener("resize", closeOnScroll);
      window.removeEventListener("scroll", closeOnScroll, true);
    };
  }, [isOpen]);

  return (
    <>
      {name && (
        <input type="hidden" name={name} value={value} disabled={disabled} />
      )}
      <button
        ref={triggerRef}
        type="button"
        role="combobox"
        aria-label={ariaLabel}
        aria-controls={isOpen ? `${id}-listbox` : undefined}
        aria-expanded={isOpen}
        aria-haspopup="listbox"
        aria-activedescendant={isOpen ? `${id}-option-${active}` : undefined}
        disabled={disabled}
        onClick={() => (isOpen ? setOpen(false) : openMenu())}
        onKeyDown={(event) => {
          if (event.key === "Escape" && isOpen) {
            event.preventDefault();
            setOpen(false);
          } else if (event.key === "Tab") {
            setOpen(false);
          } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
            event.preventDefault();
            if (!isOpen) openMenu();
            else
              setActive((index) =>
                move(index, event.key === "ArrowDown" ? 1 : -1),
              );
          } else if (event.key === "Home" && isOpen) {
            event.preventDefault();
            setActive(firstEnabled());
          } else if (event.key === "End" && isOpen) {
            event.preventDefault();
            setActive(options.findLastIndex((option) => !option.disabled));
          } else if ((event.key === "Enter" || event.key === " ") && isOpen) {
            event.preventDefault();
            choose(active);
          }
        }}
        className={cn(
          "flex min-h-10 items-center justify-between gap-3 rounded-xl border border-line bg-surface px-3 py-2.5 text-left text-sm text-foreground shadow-sm transition-colors hover:border-line-strong focus-visible:border-accent focus-visible:ring-2 focus-visible:ring-accent/30 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
          tone === "rose" &&
            "border-slate-700 bg-slate-900 text-slate-200 focus-visible:border-rose-500 focus-visible:ring-rose-500/30",
          className,
        )}
      >
        <span className={cn("min-w-0 truncate", !selected && "text-muted")}>
          {selected?.label ?? "Select an option"}
        </span>
        <ChevronDown
          className={cn(
            "h-4 w-4 shrink-0 text-muted transition-transform",
            isOpen && "rotate-180",
          )}
        />
      </button>
      {isOpen &&
        typeof document !== "undefined" &&
        createPortal(
          <div
            ref={menuRef}
            id={`${id}-listbox`}
            role="listbox"
            aria-label={ariaLabel}
            style={position}
            className={cn(
              "z-[100] overflow-y-auto rounded-xl border border-line-strong bg-canvas-raised p-1.5 shadow-2xl shadow-black/60",
              tone === "rose" && "border-slate-700 bg-slate-900",
            )}
          >
            {options.map((option, index) => (
              <div
                key={option.value}
                id={`${id}-option-${index}`}
                role="option"
                aria-selected={value === option.value}
                aria-disabled={option.disabled || undefined}
                onMouseEnter={() => !option.disabled && setActive(index)}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => choose(index)}
                className={cn(
                  "flex min-h-10 items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm text-foreground transition-colors",
                  !option.disabled && "cursor-pointer",
                  option.disabled && "cursor-not-allowed opacity-40",
                  active === index && "bg-white/8",
                  value === option.value && highlight,
                )}
              >
                <span className="min-w-0 truncate">{option.label}</span>
                {value === option.value && (
                  <Check className="h-4 w-4 shrink-0" />
                )}
              </div>
            ))}
          </div>,
          document.body,
        )}
    </>
  );
}
