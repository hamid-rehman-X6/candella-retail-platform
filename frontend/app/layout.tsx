import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Preloader } from "@/components/ui/preloader";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    default: "Candella — One platform, every retail business",
    template: "%s · Candella",
  },
  description:
    "Candella is the unified retail operating system: POS, inventory, pharmacy, garments, cosmetics, CRM and analytics — one platform that runs your entire retail business, across every store and every counter.",
  keywords: [
    "retail software",
    "point of sale",
    "POS system",
    "inventory management",
    "pharmacy ERP",
    "retail platform",
    "retail operating system",
  ],
  authors: [{ name: "Candella" }],
  openGraph: {
    title: "Candella — One platform, every retail business",
    description:
      "The unified retail operating system. POS, inventory, pharmacy, garments, cosmetics, CRM & analytics in one platform.",
    type: "website",
    siteName: "Candella",
  },
  twitter: {
    card: "summary_large_image",
    title: "Candella — One platform, every retail business",
    description:
      "The unified retail operating system for modern retail businesses.",
  },
};

export const viewport: Viewport = {
  themeColor: "#08080a",
  colorScheme: "dark",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable}`}>
      <body className="min-h-screen antialiased">
        <Preloader />
        {children}
      </body>
    </html>
  );
}
