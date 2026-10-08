import type { Metadata } from 'next';
import Link from 'next/link';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { BrowserFrame } from '@/components/browser-frame';
import { appName, gitConfig, demoConfig } from '@/lib/shared';
import { localeAlternates, socialCard } from '@/lib/metadata';
import { resolveLocale } from '@/i18n/locales';
import portfolio from '@/public/screenshots/showcase/portfolio.jpg';
import sbi from '@/public/screenshots/showcase/sbi-confirm.png';
import news from '@/public/screenshots/showcase/news-thesis.jpg';
import cashFlow from '@/public/screenshots/showcase/cash-flow.jpg';
import plan from '@/public/screenshots/showcase/plan-review.jpg';

const repo = `https://github.com/${gitConfig.user}/${gitConfig.repo}`;
const guide = (path: string) => `${repo}/blob/main/${path}`;
const features = [
  ['sbi', 'docs/features/sbi-import.md', sbi],
  ['news', 'docs/features/news-predictions.md', news],
  ['valuation', 'docs/features/analytics-metrics.md', cashFlow],
  ['plan', 'docs/showcase-demo.md', plan],
] as const;

export async function generateMetadata({ params }: { params: Promise<{ lang: string }> }): Promise<Metadata> {
  const { lang } = await params;
  const t = await getTranslations({ locale: resolveLocale(lang), namespace: 'Home' });
  const title = t('metaTitle');
  const description = t('metaDescription');
  const { images, twitter } = socialCard(title, description);
  return { title, description, alternates: localeAlternates(lang), openGraph: { title, description, siteName: appName, images }, twitter };
}

export default async function Page({ params }: { params: Promise<{ lang: string }> }) {
  const { lang } = await params;
  setRequestLocale(resolveLocale(lang));
  const t = await getTranslations('Home');
  const button = 'inline-flex rounded-lg border border-fd-border px-5 py-3 text-sm font-medium hover:bg-fd-accent focus-visible:outline-2 focus-visible:outline-offset-2';
  return (
    <div>
      <section className="mx-auto max-w-6xl space-y-7 px-6 py-16 sm:px-10">
        <p className="font-mono text-sm text-fd-muted-foreground">{t('badge')}</p>
        <h1 className="max-w-4xl text-4xl font-semibold tracking-tight sm:text-6xl">{t('heroTitle')}</h1>
        <p className="max-w-3xl text-lg leading-relaxed text-fd-muted-foreground">{t('heroSubtitle')}</p>
        <div className="flex flex-wrap gap-3">
          <Link className={`${button} bg-fd-primary text-fd-primary-foreground hover:bg-fd-primary/90`} href="#start">{t('ctaGetStarted')}</Link>
          <Link className={button} href={guide('docs/showcase-demo.md')}>{t('ctaDemo')}</Link>
          <Link className={button} href={repo}>{t('ctaGithub')}</Link>
        </div>
        <BrowserFrame image={portfolio} alt={t('portfolioAlt')} sizes="(max-width: 768px) 100vw, 1100px" priority />
        <p className="text-sm leading-relaxed text-fd-muted-foreground">{t('screenshotProvenance')}</p>
      </section>
      <section className="border-t border-fd-border px-6 py-12 sm:px-10" aria-labelledby="features-heading">
        <h2 id="features-heading" className="mb-8 text-3xl font-semibold">{t('featuresHeading')}</h2>
        <div className="grid gap-10 lg:grid-cols-2">
          {features.map(([id, path, image]) => (
            <article className="space-y-4" key={id}>
              <h3 className="text-xl font-semibold">{t(`features.${id}.title`)}</h3>
              <p className="leading-relaxed text-fd-muted-foreground">{t(`features.${id}.description`)}</p>
              <BrowserFrame image={image} alt={t(`features.${id}.alt`)} sizes="(max-width: 1024px) 100vw, 50vw" />
              <Link className="inline-block underline underline-offset-4" href={guide(path)}>{t('readGuide')}</Link>
            </article>
          ))}
        </div>
      </section>
      <section className="space-y-5 border-t border-fd-border px-6 py-12 sm:px-10">
        <h2 className="text-3xl font-semibold">{t('foundationTitle')}</h2>
        <p className="max-w-4xl leading-relaxed text-fd-muted-foreground">{t('foundationBody')}</p>
        <div className="flex flex-wrap gap-5 underline underline-offset-4">
          <Link href="https://github.com/sinhong2011/TraderMemos">TraderMemos · sinhong2011</Link>
          <Link href={guide('NOTICE')}>NOTICE</Link>
          <Link href={guide('LICENSE')}>AGPL-3.0</Link>
          <Link href={demoConfig.url}>{t('upstreamDemo')}</Link>
        </div>
      </section>
      <section id="start" className="space-y-5 border-t border-fd-border px-6 py-12 sm:px-10">
        <h2 className="text-3xl font-semibold">{t('startTitle')}</h2>
        <p className="max-w-4xl leading-relaxed text-fd-muted-foreground">{t('startBody')}</p>
        <pre className="overflow-x-auto rounded-xl bg-fd-muted p-5 text-sm"><code>{'git clone https://github.com/HoesenBruce/TradeLens.git\ncd TradeLens\ncp .env.example .env\n# Review JWT settings and pin TM_IMAGE_TAG\nmake up'}</code></pre>
        <p className="max-w-4xl leading-relaxed text-fd-muted-foreground">{t('deploymentNote')}</p>
        <div className="flex flex-wrap gap-5 underline underline-offset-4">
          <Link href={guide('README.md#quick-start')}>{t('ctaGetStarted')}</Link>
          <Link href={guide('docs/fork-deploy.md')}>{t('deployGuide')}</Link>
          <Link href={guide('docs/showcase-demo.md')}>{t('ctaDemo')}</Link>
          <Link href={`/${lang}/docs`}>{t('inheritedDocs')}</Link>
        </div>
        <p className="max-w-4xl leading-relaxed text-fd-muted-foreground">{t('mobileNote')}</p>
      </section>
    </div>
  );
}
