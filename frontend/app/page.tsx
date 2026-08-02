import { BackgroundEffects } from "@/components/landing/background-effects";
import { ScrollProgress } from "@/components/ui/scroll-progress";
import { ScrollToTop } from "@/components/ui/scroll-to-top";
import { Navbar } from "@/components/landing/navbar";
import { Hero } from "@/components/landing/hero";
import { LogoMarquee } from "@/components/landing/logo-marquee";
import { Ecosystem } from "@/components/landing/ecosystem";
import { Products } from "@/components/landing/products";
import { StackingCards } from "@/components/landing/stacking-cards";
import { Stats } from "@/components/landing/stats";
import { Industries } from "@/components/landing/industries";
import { Features } from "@/components/landing/features";
import { Testimonials } from "@/components/landing/testimonials";
import { Integrations } from "@/components/landing/integrations";
import { Pricing } from "@/components/landing/pricing";
import { Faq } from "@/components/landing/faq";
import { CTA } from "@/components/landing/cta";
import { Footer } from "@/components/landing/footer";

export default function Home() {
  return (
    <>
      <BackgroundEffects />
      <ScrollProgress />
      <Navbar />
      <main className="relative">
        <Hero />
        <LogoMarquee />
        <Ecosystem />
        <Products />
        <StackingCards />
        <Stats />
        <Industries />
        <Features />
        <Testimonials />
        <Integrations />
        <Pricing />
        <Faq />
        <CTA />
      </main>
      <Footer />
      <ScrollToTop />
    </>
  );
}
