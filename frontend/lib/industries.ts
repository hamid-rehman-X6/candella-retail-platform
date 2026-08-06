import {
  Store,
  Pill,
  Shirt,
  Sparkles,
  ShoppingCart,
  Cpu,
  Boxes,
  type LucideIcon,
} from "lucide-react";

export type Industry = { value: string; label: string; icon: LucideIcon };

/** Industry options — values match the backend tenants.industry_type CHECK. */
export const industries: Industry[] = [
  { value: "pos_general", label: "General retail", icon: Store },
  { value: "pharmacy", label: "Pharmacy", icon: Pill },
  { value: "garments", label: "Garments & fashion", icon: Shirt },
  { value: "cosmetics", label: "Cosmetics & beauty", icon: Sparkles },
  { value: "grocery", label: "Grocery & mart", icon: ShoppingCart },
  { value: "electronics", label: "Electronics", icon: Cpu },
  { value: "wholesale", label: "Wholesale", icon: Boxes },
  { value: "other", label: "Other", icon: Store },
];

export function industryLabel(value: string): string {
  return industries.find((i) => i.value === value)?.label ?? "Retail";
}
