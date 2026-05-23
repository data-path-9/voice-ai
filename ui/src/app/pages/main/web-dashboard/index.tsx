import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { ArrowUpRight } from '@carbon/icons-react';
import { Button, Dropdown, Link, Tile } from '@carbon/react';

const productFilters = ['By feature', 'Assistants', 'Deployments', 'Knowledge'];
const useCaseFilters = ['By workflow', 'Build', 'Connect', 'Monitor'];

const productCards = [
  {
    title: 'AI Assistants',
    description:
      'Build branded voice agents with prompts, tools, guardrails, and versioned provider configuration.',
    action: 'Create assistant',
    href: '/deployment/assistant/create-assistant',
  },
  {
    title: 'Deployments',
    description:
      'Publish assistants to phone calls, web widgets, API endpoints, and debugger environments.',
    action: 'Manage deployments',
    href: '/deployment/assistant',
  },
  {
    title: 'Connect your provider',
    description:
      'Connect LLM, speech-to-text, text-to-speech, storage, telemetry, and credential providers.',
    action: 'Manage providers',
    href: '/integration/models',
  },
];

const resourceCards = [
  {
    title: 'Documentation',
    description:
      'Explore product guides, API references, and setup docs for building with Rapida Voice AI.',
    action: 'View docs',
    href: 'https://doc.rapida.ai/',
    external: true,
  },
  {
    title: 'GitHub repository',
    description:
      'Review the open-source Voice AI repository, SDKs, examples, and implementation references.',
    action: 'Open GitHub',
    href: 'https://github.com/rapidaai/voice-ai',
    external: true,
  },
  {
    title: 'Pricing',
    description:
      'Compare plans and usage options for assistants, telephony, deployments, and platform features.',
    action: 'View pricing',
    href: 'https://www.rapida.ai/pricing',
    external: true,
  },
];

const newsItems = [
  {
    date: '5/21/2026',
    title: 'AgentKit and WebSocket assistant templates are ready',
    description:
      'Start from an AgentKit or WebSocket template, then configure model providers, tools, and deployment channels from one flow.',
  },
  {
    date: '5/18/2026',
    title: 'Conversation telemetry now links messages to traces',
    description:
      'Open a conversation message and inspect latency, tool calls, provider events, and execution metadata without leaving observability.',
  },
  {
    date: '5/18/2026',
    title: 'Knowledge connectors support cloud document sources',
    description:
      'Connect Google Drive, OneDrive, SharePoint, Confluence, GitHub, and Notion sources to keep assistant knowledge current.',
  },
];

export const HomePage = () => {
  const navigation = useGlobalNavigation();
  const { user } = useCurrentCredential();
  const firstName = user?.name?.trim().split(/\s+/)[0] || 'Prashant';

  return (
    <div className="min-h-0 flex-1 overflow-auto bg-white px-4 py-5 text-[#161616] dark:bg-[#161616] dark:text-[#f4f4f4] md:px-6">
      <div className="mx-auto flex max-w-[1500px] flex-col gap-5">
        <h1 className="text-[1.75rem] font-semibold leading-tight tracking-normal">
          Welcome, {firstName}!
        </h1>

        <Tile className="relative !min-h-[180px] !overflow-hidden !rounded-none !border-0 !bg-primary/10 !p-0 dark:!bg-primary/10">
          <div className="absolute inset-0 bg-[linear-gradient(100deg,color-mix(in_oklab,var(--color-primary)_12%,white)_0%,color-mix(in_oklab,var(--color-primary)_8%,white)_44%,color-mix(in_oklab,var(--color-primary)_22%,white)_72%,color-mix(in_oklab,var(--color-primary)_42%,white)_100%)] dark:bg-[linear-gradient(100deg,color-mix(in_oklab,var(--color-primary)_22%,#161616)_0%,color-mix(in_oklab,var(--color-primary)_16%,#161616)_48%,color-mix(in_oklab,var(--color-primary)_30%,#161616)_100%)]" />
          <div className="absolute right-0 top-0 hidden h-full w-[58%] overflow-hidden md:block">
            <div className="absolute right-[18%] top-[-28%] h-[160%] w-[34%] rotate-[22deg] bg-primary/10 dark:bg-white/10" />
            <div className="absolute right-[38%] top-[-20%] h-[150%] w-[26%] rotate-[22deg] bg-primary/10 dark:bg-white/10" />
            <div className="absolute right-0 top-0 h-full w-[18%] bg-primary/10 dark:bg-white/10" />
          </div>
          <div className="relative z-[1] max-w-[620px] px-8 py-8">
            <h2 className="text-lg font-semibold leading-6">
              Product Highlight: Voice AI Assistants
            </h2>
            <p className="mt-4 text-base leading-6 text-[#262626] dark:text-[#f4f4f4]">
              Design, ground, deploy, and monitor real-time AI assistants across
              voice, web, API, and debugger channels from one workspace.
            </p>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              <Button
                size="md"
                kind="primary"
                onClick={() => navigation.goToCreateAssistant()}
              >
                Create first voice agent
              </Button>
              <Button
                size="md"
                kind="secondary"
                href="https://cal.com/prashant-srivastav-u8duzh/30min"
                target="_blank"
                rel="noopener noreferrer"
              >
                Talk to us
              </Button>
            </div>
          </div>
        </Tile>

        <div className="grid grid-cols-1 gap-8 xl:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
          <main className="min-w-0">
            <div className="mb-4 flex flex-wrap items-center gap-2">
              <h2 className="mr-1 text-lg font-semibold">Explore Features</h2>
              <Dropdown
                id="dashboard-product-filter"
                hideLabel
                titleText=""
                label="By product"
                size="sm"
                type="inline"
                items={productFilters}
                selectedItem={productFilters[0]}
                className="dashboard-filter-dropdown min-w-[120px]"
              />
              <Dropdown
                id="dashboard-use-case-filter"
                hideLabel
                titleText=""
                label="By use-case"
                size="sm"
                type="inline"
                items={useCaseFilters}
                selectedItem={useCaseFilters[0]}
                className="dashboard-filter-dropdown min-w-[132px]"
              />
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3 md:auto-rows-fr">
              {productCards.map(card => (
                <Tile
                  key={card.title}
                  className="!flex !h-full !min-h-[184px] !flex-col !rounded-none !border !border-[#e0e0e0] !bg-white !p-6 dark:!border-[#393939] dark:!bg-[#262626]"
                >
                  <h3 className="text-base font-semibold">{card.title}</h3>
                  <p className="mt-4 flex-1 text-sm leading-5 text-[#393939] dark:text-[#c6c6c6]">
                    {card.description}
                  </p>
                  <div className="mt-auto pt-6">
                    <Button size="sm" kind="tertiary" href={card.href}>
                      {card.action}
                    </Button>
                  </div>
                </Tile>
              ))}
            </div>

            <h2 className="mb-4 mt-6 text-lg font-semibold">
              Help and Resources
            </h2>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3 md:auto-rows-fr">
              {resourceCards.map(card => (
                <Tile
                  key={card.title}
                  className="!flex !h-full !min-h-[204px] !flex-col !rounded-none !border !border-[#e0e0e0] !bg-white !p-6 dark:!border-[#393939] dark:!bg-[#262626]"
                >
                  <h3 className="text-base font-semibold">{card.title}</h3>
                  <p className="mt-4 flex-1 text-sm leading-5 text-[#393939] dark:text-[#c6c6c6]">
                    {card.description}
                  </p>
                  <div className="mt-auto pt-6">
                    <Link
                      href={card.href}
                      target={card.external ? '_blank' : undefined}
                      rel={card.external ? 'noopener noreferrer' : undefined}
                      className="!inline-flex !items-center !gap-1 !text-sm"
                    >
                      {card.action}
                      {card.external && <ArrowUpRight size={12} />}
                    </Link>
                  </div>
                </Tile>
              ))}
            </div>
          </main>

          <aside className="min-w-0">
            <h2 className="mb-4 text-lg font-semibold">What's new</h2>
            <Tile className="!flex !min-h-[446px] !flex-col !rounded-none !border-0 !bg-[#e8e8e8] !p-7 dark:!bg-[#262626]">
              <div className="flex flex-col gap-8">
                {newsItems.map(item => (
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
                      href="/observability/conversation"
                      className="mt-2 !inline-flex !items-center !gap-1 !text-sm"
                    >
                      Read more <ArrowUpRight size={12} />
                    </Link>
                  </article>
                ))}
              </div>
            </Tile>
          </aside>
        </div>
      </div>
    </div>
  );
};
