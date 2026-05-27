import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import {
  GhostButton,
  PrimaryButton,
  SecondaryButton,
  TertiaryButton,
} from '@/app/components/carbon/button';
import { ArrowUpRight } from '@carbon/icons-react';
import { Link, SkeletonPlaceholder, SkeletonText, Tile } from '@carbon/react';
import { useEffect, useState, type ReactNode } from 'react';

const DASHBOARD_DESIGN_URL =
  'https://cdn-01.rapida.ai/web/rapida-dashboard-v1.json';

type DashboardButtonKind = 'primary' | 'secondary' | 'tertiary' | 'ghost';

type DashboardHeroAction = {
  label: string;
  kind: DashboardButtonKind;
  intent?: 'createAssistant';
  href?: string;
  external?: boolean;
};

type DashboardCard = {
  title: string;
  description: string;
  action: string;
  href: string;
  external?: boolean;
};

type DashboardSection = {
  title: string;
  layout: 'feature-grid' | 'resource-grid';
  cards: DashboardCard[];
};

type DashboardNewsItem = {
  date: string;
  title: string;
  description: string;
};

type DashboardDesign = {
  welcome: {
    prefix: string;
    fallbackName: string;
  };
  hero: {
    title: string;
    description: string;
    actions: DashboardHeroAction[];
  };
  sections: DashboardSection[];
  news: {
    title: string;
    readMoreHref: string;
    items: DashboardNewsItem[];
  };
};

const dashboardButtonKinds: DashboardButtonKind[] = [
  'primary',
  'secondary',
  'tertiary',
  'ghost',
];

const dashboardSectionLayouts: DashboardSection['layout'][] = [
  'feature-grid',
  'resource-grid',
];

const dashboardHeroActionButtonByKind: Record<
  DashboardButtonKind,
  typeof PrimaryButton
> = {
  primary: PrimaryButton,
  secondary: SecondaryButton,
  tertiary: TertiaryButton,
  ghost: GhostButton,
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isString(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0;
}

function isOptionalBoolean(value: unknown): value is boolean | undefined {
  return value === undefined || typeof value === 'boolean';
}

function isOptionalString(value: unknown): value is string | undefined {
  return value === undefined || isString(value);
}

function isDashboardButtonKind(value: unknown): value is DashboardButtonKind {
  return (
    typeof value === 'string' &&
    dashboardButtonKinds.includes(value as DashboardButtonKind)
  );
}

function isDashboardHeroAction(value: unknown): value is DashboardHeroAction {
  if (!isRecord(value)) return false;

  return (
    isString(value.label) &&
    isDashboardButtonKind(value.kind) &&
    (value.intent === undefined || value.intent === 'createAssistant') &&
    isOptionalString(value.href) &&
    isOptionalBoolean(value.external)
  );
}

function isDashboardCard(value: unknown): value is DashboardCard {
  if (!isRecord(value)) return false;

  return (
    isString(value.title) &&
    isString(value.description) &&
    isString(value.action) &&
    isString(value.href) &&
    isOptionalBoolean(value.external)
  );
}

function isDashboardSection(value: unknown): value is DashboardSection {
  if (!isRecord(value)) return false;

  return (
    isString(value.title) &&
    typeof value.layout === 'string' &&
    dashboardSectionLayouts.includes(
      value.layout as DashboardSection['layout'],
    ) &&
    Array.isArray(value.cards) &&
    value.cards.every(isDashboardCard)
  );
}

function isDashboardNewsItem(value: unknown): value is DashboardNewsItem {
  if (!isRecord(value)) return false;

  return (
    isString(value.date) && isString(value.title) && isString(value.description)
  );
}

function isDashboardDesign(value: unknown): value is DashboardDesign {
  if (!isRecord(value)) return false;

  const { welcome, hero, sections, news } = value;

  return (
    isRecord(welcome) &&
    isString(welcome.prefix) &&
    isString(welcome.fallbackName) &&
    isRecord(hero) &&
    isString(hero.title) &&
    isString(hero.description) &&
    Array.isArray(hero.actions) &&
    hero.actions.every(isDashboardHeroAction) &&
    Array.isArray(sections) &&
    sections.every(isDashboardSection) &&
    isRecord(news) &&
    isString(news.title) &&
    isString(news.readMoreHref) &&
    Array.isArray(news.items) &&
    news.items.every(isDashboardNewsItem)
  );
}

function DashboardFrame({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-0 flex-1 overflow-auto bg-white px-4 py-5 text-[#161616] dark:bg-[#161616] dark:text-[#f4f4f4] md:px-6">
      <div className="mx-auto flex max-w-[1500px] flex-col gap-5">
        {children}
      </div>
    </div>
  );
}

function Title({
  isLoading,
  welcomePrefix,
  firstName,
}: {
  isLoading: boolean;
  welcomePrefix?: string;
  firstName?: string;
}) {
  if (isLoading) {
    return <SkeletonText heading width="280px" className="!mb-1" />;
  }

  return (
    <h1 className="text-[1.75rem] font-semibold leading-tight tracking-normal">
      {welcomePrefix}, {firstName}!
    </h1>
  );
}

function Hero({
  isLoading,
  hero,
  onHeroAction,
}: {
  isLoading: boolean;
  hero?: DashboardDesign['hero'];
  onHeroAction: (action: DashboardHeroAction) => void;
}) {
  if (isLoading) {
    return (
      <Tile className="min-h-[180px]! overflow-hidden! rounded-none! border-0! bg-primary/10! p-8! dark:bg-primary/10!">
        <div className="max-w-[620px]">
          <SkeletonText heading width="72%" className="!mb-3" />
          <SkeletonText
            paragraph
            lineCount={3}
            width="100%"
            className="!mb-6"
          />
          <div className="flex flex-wrap gap-3">
            <SkeletonPlaceholder className="!h-10 !w-[220px]" />
            <SkeletonPlaceholder className="!h-10 !w-[148px]" />
          </div>
        </div>
      </Tile>
    );
  }

  if (!hero) return null;

  return (
    <Tile className="relative min-h-[180px]! overflow-hidden! rounded-none! border-0! bg-primary/10! p-0! dark:bg-primary/10!">
      <div className="absolute inset-0 bg-[linear-gradient(100deg,color-mix(in_oklab,var(--color-primary)_12%,white)_0%,color-mix(in_oklab,var(--color-primary)_8%,white)_44%,color-mix(in_oklab,var(--color-primary)_22%,white)_72%,color-mix(in_oklab,var(--color-primary)_42%,white)_100%)] dark:bg-[linear-gradient(100deg,color-mix(in_oklab,var(--color-primary)_22%,#161616)_0%,color-mix(in_oklab,var(--color-primary)_16%,#161616)_48%,color-mix(in_oklab,var(--color-primary)_30%,#161616)_100%)]" />
      <div className="absolute right-0 top-0 hidden h-full w-[58%] overflow-hidden md:block">
        <div className="absolute right-[18%] top-[-28%] h-[160%] w-[34%] rotate-[22deg] bg-primary/10 dark:bg-white/10" />
        <div className="absolute right-[38%] top-[-20%] h-[150%] w-[26%] rotate-[22deg] bg-primary/10 dark:bg-white/10" />
        <div className="absolute right-0 top-0 h-full w-[18%] bg-primary/10 dark:bg-white/10" />
      </div>
      <div className="relative z-[1] max-w-[620px] px-8 py-8">
        <h2 className="text-lg font-semibold leading-6">{hero.title}</h2>
        <p className="mt-4 text-base leading-6 text-[#262626] dark:text-[#f4f4f4]">
          {hero.description}
        </p>
        <div className="mt-6 flex flex-wrap items-center gap-3">
          {hero.actions.map(action => {
            const HeroActionButton =
              dashboardHeroActionButtonByKind[action.kind];

            return (
              <HeroActionButton
                key={action.label}
                size="md"
                href={action.href}
                target={action.external ? '_blank' : undefined}
                rel={action.external ? 'noopener noreferrer' : undefined}
                onClick={action.intent ? () => onHeroAction(action) : undefined}
              >
                {action.label}
              </HeroActionButton>
            );
          })}
        </div>
      </div>
    </Tile>
  );
}

function FeatureCard({
  isLoading,
  layout,
  card,
}: {
  isLoading: boolean;
  layout: DashboardSection['layout'];
  card?: DashboardCard;
}) {
  if (isLoading || !card) {
    return (
      <Tile className="flex! min-h-[184px]! flex-col! rounded-none! border! border-[#e0e0e0]! bg-white! p-6! dark:border-[#393939]! dark:bg-[#262626]!">
        <SkeletonText heading width="70%" className="!mb-4" />
        <SkeletonText paragraph lineCount={3} width="100%" className="!mb-6" />
        <SkeletonPlaceholder className="!mt-auto !h-8 !w-[140px]" />
      </Tile>
    );
  }

  return (
    <Tile className="flex! h-full! min-h-[184px]! flex-col! rounded-none! border! border-[#e0e0e0]! bg-white! p-6! dark:border-[#393939]! dark:bg-[#262626]!">
      <h3 className="text-base font-semibold">{card.title}</h3>
      <p className="mt-4 flex-1 text-sm leading-5 text-[#393939] dark:text-[#c6c6c6]">
        {card.description}
      </p>
      <div className="mt-auto pt-6">
        {layout === 'feature-grid' ? (
          <TertiaryButton size="sm" href={card.href}>
            {card.action}
          </TertiaryButton>
        ) : (
          <Link
            href={card.href}
            target={card.external ? '_blank' : undefined}
            rel={card.external ? 'noopener noreferrer' : undefined}
            className="inline-flex! items-center! gap-1! text-sm!"
          >
            {card.action}
            {card.external && <ArrowUpRight size={12} />}
          </Link>
        )}
      </div>
    </Tile>
  );
}

function SectionCards({
  isLoading,
  section,
  className,
}: {
  isLoading: boolean;
  section?: DashboardSection;
  className?: string;
}) {
  if (isLoading) {
    return (
      <section className={className}>
        <SkeletonText heading width="220px" className="!mb-4" />
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3 md:auto-rows-fr">
          {[0, 1, 2].map(cardIndex => (
            <FeatureCard
              key={cardIndex}
              isLoading
              layout={section?.layout ?? 'feature-grid'}
            />
          ))}
        </div>
      </section>
    );
  }

  if (!section) return null;

  return (
    <section className={className}>
      <h2 className="mb-4 text-lg font-semibold">{section.title}</h2>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3 md:auto-rows-fr">
        {section.cards.map(card => (
          <FeatureCard
            key={card.title}
            isLoading={false}
            layout={section.layout}
            card={card}
          />
        ))}
      </div>
    </section>
  );
}

function HelpAndResource({
  isLoading,
  section,
}: {
  isLoading: boolean;
  section?: DashboardSection;
}) {
  return (
    <SectionCards isLoading={isLoading} section={section} className="mt-6" />
  );
}

function News({
  isLoading,
  news,
}: {
  isLoading: boolean;
  news?: DashboardDesign['news'];
}) {
  if (isLoading) {
    return (
      <>
        <SkeletonText heading width="170px" className="!mb-4" />
        <Tile className="min-h-[446px]! rounded-none! border-0! bg-[#e8e8e8]! p-7! dark:bg-[#262626]!">
          <div className="flex flex-col gap-8">
            {[0, 1, 2].map(item => (
              <article key={`news-${item}`}>
                <SkeletonText width="90px" className="!mb-3" />
                <SkeletonText heading width="86%" className="!mb-2" />
                <SkeletonText
                  paragraph
                  lineCount={2}
                  width="100%"
                  className="!mb-3"
                />
                <SkeletonPlaceholder className="!h-4 !w-[84px]" />
              </article>
            ))}
          </div>
        </Tile>
      </>
    );
  }

  if (!news) return null;

  return (
    <>
      <h2 className="mb-4 text-lg font-semibold">{news.title}</h2>
      <Tile className="flex min-h-[446px] flex-col rounded-none border-0 bg-[#e8e8e8] p-7 dark:bg-[#262626]">
        <div className="flex flex-col gap-8">
          {news.items.map(item => (
            <article key={`${item.date}-${item.title}`}>
              <p className="text-xs font-medium text-[#525252] dark:text-[#c6c6c6]">
                {item.date}
              </p>
              <h3 className="mt-2 text-sm font-semibold leading-5">
                {item.title}
              </h3>
              <p className="mt-2 text-sm leading-5 text-[#393939] dark:text-[#c6c6c6]">
                {item.description}
              </p>
              <Link
                href={news.readMoreHref}
                className="mt-3! inline-flex! items-center! gap-1! text-sm!"
              >
                Read more <ArrowUpRight size={12} />
              </Link>
            </article>
          ))}
        </div>
      </Tile>
    </>
  );
}

export const HomePage = () => {
  const navigation = useGlobalNavigation();
  const { user } = useCurrentCredential();
  const [dashboardDesign, setDashboardDesign] =
    useState<DashboardDesign | null>(null);
  const [hasLoadError, setHasLoadError] = useState(false);

  const handleHeroAction = (action: DashboardHeroAction) => {
    if (action.intent === 'createAssistant') {
      navigation.goToCreateAssistant();
    }
  };

  useEffect(() => {
    let cancelled = false;
    fetch(DASHBOARD_DESIGN_URL, { cache: 'no-store' })
      .then(response => {
        if (!response.ok) throw new Error('Unable to load dashboard design');
        return response.json();
      })
      .then(data => {
        if (!isDashboardDesign(data)) {
          throw new Error('Invalid dashboard design');
        }

        if (!cancelled) {
          setHasLoadError(false);
          setDashboardDesign(data);
        }
      })
      .catch(() => {
        if (!cancelled) setHasLoadError(true);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const isLoading = !dashboardDesign && !hasLoadError;
  const featureSection = dashboardDesign?.sections.find(
    section => section.layout === 'feature-grid',
  );
  const helpAndResourceSection = dashboardDesign?.sections.find(
    section => section.layout === 'resource-grid',
  );

  if (hasLoadError && !dashboardDesign) {
    return (
      <DashboardFrame>
        <Tile className="min-h-[180px]! rounded-none! border! border-[#e0e0e0]! bg-white! p-6! dark:border-[#393939]! dark:bg-[#262626]!">
          <h1 className="text-[1.75rem] font-semibold leading-tight tracking-normal">
            Dashboard unavailable
          </h1>
          <p className="mt-4 max-w-[620px] text-sm leading-5 text-[#525252] dark:text-[#c6c6c6]">
            The dashboard design could not be loaded.
          </p>
        </Tile>
      </DashboardFrame>
    );
  }

  const firstName =
    user?.name?.trim().split(/\s+/)[0] || dashboardDesign?.welcome.fallbackName;

  return (
    <DashboardFrame>
      <Title
        isLoading={isLoading}
        welcomePrefix={dashboardDesign?.welcome.prefix}
        firstName={firstName}
      />
      <Hero
        isLoading={isLoading}
        hero={dashboardDesign?.hero}
        onHeroAction={handleHeroAction}
      />

      <div className="grid grid-cols-1 gap-8 xl:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
        <main className="min-w-0">
          <SectionCards isLoading={isLoading} section={featureSection} />
          <HelpAndResource
            isLoading={isLoading}
            section={helpAndResourceSection}
          />
        </main>
        <aside className="min-w-0">
          <News isLoading={isLoading} news={dashboardDesign?.news} />
        </aside>
      </div>
    </DashboardFrame>
  );
};
