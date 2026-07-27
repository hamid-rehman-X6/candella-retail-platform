import {
  Store,
  Pill,
  Shirt,
  Sparkles,
  Boxes,
  Users,
  BarChart3,
  CreditCard,
  Cloud,
  ShieldCheck,
  Zap,
  Wifi,
  Receipt,
  Package,
  LineChart,
  Bell,
  Globe,
  RefreshCw,
  Layers,
  Smartphone,
  type LucideIcon,
} from "lucide-react";

/* ------------------------------------------------------------------ *
 * Navigation
 * ------------------------------------------------------------------ */
export const navLinks = [
  { label: "Platform", href: "#ecosystem" },
  { label: "Products", href: "#products" },
  { label: "Industries", href: "#industries" },
  { label: "Features", href: "#features" },
  { label: "Pricing", href: "#pricing" },
  { label: "FAQ", href: "#faq" },
] as const;

/* ------------------------------------------------------------------ *
 * Product modules — the core of the ecosystem
 * ------------------------------------------------------------------ */
export type ProductModule = {
  id: string;
  name: string;
  tagline: string;
  description: string;
  icon: LucideIcon;
  accent: string; // tailwind gradient stops
  features: string[];
};

export const productModules: ProductModule[] = [
  {
    id: "pos",
    name: "Candella POS",
    tagline: "Sell anywhere, in seconds",
    description:
      "A lightning-fast point of sale that works on any device, online or off. Ring up sales, split payments, apply discounts and print or email receipts without missing a beat.",
    icon: Store,
    accent: "from-amber-400/25 to-orange-500/10",
    features: [
      "Offline-first, syncs when back online",
      "Barcode & QR scanning",
      "Split & partial payments",
      "Cash-drawer reconciliation",
    ],
  },
  {
    id: "inventory",
    name: "Inventory & Warehouse",
    tagline: "Never oversell again",
    description:
      "Real-time stock across every store and warehouse, with batch, expiry and serial tracking, transfers, purchase orders and automated reorder points built in.",
    icon: Boxes,
    accent: "from-amber-300/25 to-yellow-500/10",
    features: [
      "Multi-location stock in real time",
      "Batch, lot & expiry tracking",
      "Automated reorder alerts",
      "Purchase orders & receiving",
    ],
  },
  {
    id: "pharmacy",
    name: "Pharmacy ERP",
    tagline: "Compliance, built in",
    description:
      "Purpose-built for pharmacies: prescription handling, drug schedules, near-expiry control and supplier management that keeps you compliant and stocked.",
    icon: Pill,
    accent: "from-orange-400/25 to-amber-500/10",
    features: [
      "Prescription & schedule tracking",
      "Near-expiry & recall control",
      "Supplier & distributor ledgers",
      "Regulatory-ready reporting",
    ],
  },
  {
    id: "garments",
    name: "Garments & Fashion",
    tagline: "Every size, every season",
    description:
      "Matrix inventory for size and color, season and collection planning, and style-level margins so fashion retailers move the right stock at the right time.",
    icon: Shirt,
    accent: "from-yellow-400/25 to-amber-500/10",
    features: [
      "Size / color variant matrix",
      "Season & collection planning",
      "Style-level margin analytics",
      "Barcode label printing",
    ],
  },
  {
    id: "cosmetics",
    name: "Cosmetics & Beauty",
    tagline: "Shades, batches & shelf life",
    description:
      "Track shades and SKUs, manage batch shelf life, and run loyalty and bundle promotions that keep beauty customers coming back counter after counter.",
    icon: Sparkles,
    accent: "from-amber-400/25 to-rose-500/10",
    features: [
      "Shade & variant catalog",
      "Batch shelf-life alerts",
      "Bundles & gift-with-purchase",
      "Loyalty & tiered rewards",
    ],
  },
  {
    id: "crm",
    name: "Customers & CRM",
    tagline: "Know every shopper",
    description:
      "A unified customer profile across every store and channel — purchase history, loyalty points, segments and targeted campaigns that lift repeat revenue.",
    icon: Users,
    accent: "from-amber-300/25 to-orange-400/10",
    features: [
      "Unified customer profiles",
      "Loyalty points & store credit",
      "Segments & campaigns",
      "Purchase history & insights",
    ],
  },
];

/* ------------------------------------------------------------------ *
 * Ecosystem pillars — the "why one platform" story
 * ------------------------------------------------------------------ */
export const ecosystemPillars = [
  {
    icon: Layers,
    title: "One shared core",
    description:
      "Products, stock, customers and payments live in a single system — every module reads and writes the same source of truth. No syncing, no silos.",
  },
  {
    icon: RefreshCw,
    title: "Real-time everywhere",
    description:
      "A sale on the shop floor updates head-office dashboards, warehouse stock and reorder points the instant it happens.",
  },
  {
    icon: LineChart,
    title: "Grows with you",
    description:
      "Start with a single counter and scale to hundreds of stores across industries — without ever replatforming.",
  },
];

/* ------------------------------------------------------------------ *
 * Stacking-cards deep dive — "how it flows"
 * ------------------------------------------------------------------ */
export type FlowCard = {
  step: string;
  title: string;
  description: string;
  icon: LucideIcon;
  points: string[];
};

export const flowCards: FlowCard[] = [
  {
    step: "01",
    title: "Sell at the counter",
    description:
      "Your team rings up a sale on Candella POS — on a tablet, desktop or dedicated terminal. It works even when the internet drops.",
    icon: Store,
    points: ["Any device", "Works offline", "Sub-second checkout"],
  },
  {
    step: "02",
    title: "Stock updates instantly",
    description:
      "The moment a sale completes, inventory across every location adjusts in real time and reorder points recalculate automatically.",
    icon: Boxes,
    points: [
      "Live multi-store stock",
      "Auto reorder points",
      "Batch & expiry aware",
    ],
  },
  {
    step: "03",
    title: "Customers are remembered",
    description:
      "Loyalty points accrue, the purchase joins a unified customer profile, and segments update for your next campaign — no manual work.",
    icon: Users,
    points: ["Unified profiles", "Loyalty & credit", "Auto segmentation"],
  },
  {
    step: "04",
    title: "Leaders see everything",
    description:
      "Head-office dashboards reflect revenue, margin and stock health across the whole business the instant it changes — one screen, every store.",
    icon: BarChart3,
    points: ["Live dashboards", "Margin & COGS", "Every store, one view"],
  },
];

/* ------------------------------------------------------------------ *
 * Industries
 * ------------------------------------------------------------------ */
export const industries = [
  {
    icon: Pill,
    name: "Pharmacy",
    blurb: "Prescriptions, schedules & expiry control.",
  },
  {
    icon: Shirt,
    name: "Garments & Fashion",
    blurb: "Size/color matrix & season planning.",
  },
  {
    icon: Sparkles,
    name: "Cosmetics & Beauty",
    blurb: "Shades, batches & loyalty programs.",
  },
  {
    icon: Store,
    name: "Grocery & Mart",
    blurb: "High-volume checkout & weighing scales.",
  },
  {
    icon: Package,
    name: "Electronics",
    blurb: "Serial numbers, warranties & bundles.",
  },
  {
    icon: Boxes,
    name: "Wholesale & Distribution",
    blurb: "Bulk pricing, credit & ledgers.",
  },
];

/* ------------------------------------------------------------------ *
 * Feature bento
 * ------------------------------------------------------------------ */
export type Feature = {
  icon: LucideIcon;
  title: string;
  description: string;
  span?: "wide" | "tall" | "normal";
};

export const features: Feature[] = [
  {
    icon: Wifi,
    title: "Offline-first by design",
    description:
      "Keep selling through internet outages. Every transaction is queued locally and synced automatically the moment you reconnect.",
    span: "wide",
  },
  {
    icon: BarChart3,
    title: "Live analytics",
    description:
      "Revenue, margin and stock health across every store, updated in real time.",
  },
  {
    icon: ShieldCheck,
    title: "Enterprise-grade security",
    description:
      "Role-based access, full audit trails and row-level tenant isolation.",
  },
  {
    icon: CreditCard,
    title: "Any payment method",
    description:
      "Cash, card, wallets, bank transfer and store credit — split across a single sale.",
  },
  {
    icon: Cloud,
    title: "Cloud + edge",
    description: "Central data in the cloud, instant response at the counter.",
  },
  {
    icon: Receipt,
    title: "Smart receipts",
    description: "Print, email or WhatsApp branded receipts with every sale.",
  },
  {
    icon: Bell,
    title: "Proactive alerts",
    description:
      "Low stock, near-expiry and unusual activity — before they cost you.",
    span: "wide",
  },
  {
    icon: Smartphone,
    title: "Mobile ready",
    description: "Run the whole business from a phone or tablet.",
  },
];

/* ------------------------------------------------------------------ *
 * Stats
 * ------------------------------------------------------------------ */
export type Stat = {
  value: number;
  label: string;
  prefix?: string;
  suffix?: string;
  decimals?: number;
};

export const stats: Stat[] = [
  { value: 6, suffix: "+", label: "Retail verticals, one platform" },
  {
    value: 99.9,
    suffix: "%",
    label: "Uptime, offline-first at the edge",
    decimals: 1,
  },
  { value: 40, suffix: "%", label: "Faster checkout on average" },
  { value: 1, prefix: "<", suffix: "s", label: "Sub-second sale sync" },
];

/* ------------------------------------------------------------------ *
 * Testimonials
 * ------------------------------------------------------------------ */
export const testimonials = [
  {
    quote:
      "We replaced four disconnected tools with Candella. Stock, sales and customers finally live in one place — our month-end close went from days to hours.",
    name: "Ayesha Khan",
    role: "Operations Director, MediCare Pharmacy Group",
  },
  {
    quote:
      "The offline mode alone paid for itself. When our internet went down on the busiest day of the year, checkout never stopped for a second.",
    name: "Daniel Osei",
    role: "Owner, Thread & Co. Fashion",
  },
  {
    quote:
      "Rolling out to 30 stores was painless. New locations are live the same day, and head office sees every counter on one dashboard.",
    name: "Priya Nair",
    role: "CEO, Glow Beauty Retail",
  },
  {
    quote:
      "Batch and expiry tracking is flawless. We cut expired-stock write-offs by more than half in the first quarter.",
    name: "Marcus Feld",
    role: "Inventory Lead, CityMart Grocery",
  },
];

/* ------------------------------------------------------------------ *
 * Pricing
 * ------------------------------------------------------------------ */
export type PricingTier = {
  name: string;
  price: string;
  period: string;
  description: string;
  features: string[];
  cta: string;
  featured?: boolean;
};

export const pricingTiers: PricingTier[] = [
  {
    name: "Starter",
    price: "$29",
    period: "/store / month",
    description: "For single-counter shops getting started.",
    features: [
      "1 store, 2 registers",
      "Candella POS + Inventory",
      "Up to 2,000 products",
      "Email support",
    ],
    cta: "Start free trial",
  },
  {
    name: "Growth",
    price: "$79",
    period: "/store / month",
    description: "For multi-store retailers scaling up.",
    features: [
      "Unlimited registers",
      "All industry modules",
      "CRM, loyalty & campaigns",
      "Advanced analytics",
      "Priority support",
    ],
    cta: "Start free trial",
    featured: true,
  },
  {
    name: "Enterprise",
    price: "Custom",
    period: "tailored to you",
    description: "For chains & franchises with bespoke needs.",
    features: [
      "Unlimited stores",
      "White-label & SSO",
      "Dedicated success manager",
      "Custom integrations & SLA",
    ],
    cta: "Talk to sales",
  },
];

/* ------------------------------------------------------------------ *
 * Integrations (marquee)
 * ------------------------------------------------------------------ */
export const integrations = [
  "Stripe",
  "Easypaisa",
  "JazzCash",
  "PayTabs",
  "Moyasar",
  "QuickBooks",
  "Shopify",
  "WhatsApp",
  "Xero",
  "Cloudflare",
  "Twilio",
  "Google Analytics",
];

/* ------------------------------------------------------------------ *
 * FAQ
 * ------------------------------------------------------------------ */
export const faqs = [
  {
    q: "What exactly is Candella?",
    a: "Candella is a single retail operating system that runs your whole business — point of sale, inventory, customers, payments and analytics — with industry-specific modules for pharmacy, garments, cosmetics and more. One platform instead of a patchwork of disconnected tools.",
  },
  {
    q: "Does it really work offline?",
    a: "Yes. Candella POS is offline-first: sales, stock changes and receipts are stored locally on the device and sync automatically the moment your connection returns. Checkout never stops, even during an internet outage.",
  },
  {
    q: "Can I use it across multiple stores?",
    a: "Absolutely. Candella is built for multi-location retail from day one. Stock, pricing, customers and reporting are shared across every store in real time, while each location keeps its own registers and staff.",
  },
  {
    q: "Which industries do you support?",
    a: "Any retail business — with purpose-built modules for pharmacy, garments & fashion, cosmetics & beauty, grocery, electronics and wholesale. The shared core means you can run several verticals under one roof.",
  },
  {
    q: "How long does it take to get started?",
    a: "Most single stores are live the same day. Import your products, set up your registers and start selling. Larger rollouts are guided by our team, with new locations typically live within hours.",
  },
  {
    q: "Is my data secure?",
    a: "Security is foundational: role-based access control, complete audit trails, encrypted data and strict tenant isolation so your business data is never mixed with anyone else's.",
  },
];

/* ------------------------------------------------------------------ *
 * Footer
 * ------------------------------------------------------------------ */
export const footerColumns = [
  {
    title: "Product",
    links: [
      { label: "Point of Sale", href: "#products" },
      { label: "Inventory", href: "#products" },
      { label: "Pharmacy ERP", href: "#products" },
      { label: "CRM & Loyalty", href: "#products" },
      { label: "Analytics", href: "#features" },
    ],
  },
  {
    title: "Industries",
    links: [
      { label: "Pharmacy", href: "#industries" },
      { label: "Garments", href: "#industries" },
      { label: "Cosmetics", href: "#industries" },
      { label: "Grocery", href: "#industries" },
      { label: "Wholesale", href: "#industries" },
    ],
  },
  {
    title: "Company",
    links: [
      { label: "About", href: "#" },
      { label: "Careers", href: "#" },
      { label: "Blog", href: "#" },
      { label: "Contact", href: "#" },
    ],
  },
  {
    title: "Resources",
    links: [
      { label: "Documentation", href: "#" },
      { label: "Pricing", href: "#pricing" },
      { label: "Support", href: "#" },
      { label: "Status", href: "#" },
    ],
  },
];

export const marqueeLogos = [
  "MediCare",
  "Thread & Co.",
  "Glow Beauty",
  "CityMart",
  "PharmaPlus",
  "Urban Threads",
  "Lumière",
  "FreshFold",
];

export const trustFeatures = [
  { icon: Zap, label: "Sub-second checkout" },
  { icon: Wifi, label: "Works offline" },
  { icon: Globe, label: "Multi-store & multi-currency" },
  { icon: LineChart, label: "Real-time analytics" },
];
