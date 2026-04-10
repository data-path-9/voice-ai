import { Helmet } from '@/app/components/helmet';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { IconOnlyButton } from '@/app/components/carbon/button';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  Renew,
  View,
  Edit,
  Launch,
  Copy,
  Checkmark,
  Microphone,
  VolumeUp,
  Settings,
  Phone,
  Deploy,
} from '@carbon/icons-react';
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { useParams } from 'react-router-dom';
import {
  Assistant,
  AssistantDefinition,
  ConnectionConfig,
  GetAssistant,
  GetAssistantRequest,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { connectionConfig } from '@/configs';
import { useRapidaStore } from '@/hooks';
import { toHumanReadableDateTime } from '@/utils/date';
import { AssistantPhoneCallDeploymentDialog } from '@/app/components/base/modal/assistant-phone-call-deployment-modal';
import { AssistantDebugDeploymentDialog } from '@/app/components/base/modal/assistant-debug-deployment-modal';
import { DeploymentEditSectionModal } from '@/app/components/base/modal/assistant-debugger-edit-section-modal';
import { AssistantWebWidgetlDeploymentDialog } from '@/app/components/base/modal/assistant-web-widget-deployment-modal';
import { AssistantApiDeploymentDialog } from '@/app/components/base/modal/assistant-api-deployment-modal';
import SourceIndicator from '@/app/components/indicators/source';
import { ConfigureExperienceModalForm } from '@/app/components/base/modal/assistant-debugger-edit-section-modal/configure-experience-form';
import { ConfigureWebExperienceModalForm } from '@/app/components/base/modal/assistant-debugger-edit-section-modal/configure-web-experience-form';
import { ConfigureAudioInputModalForm } from '@/app/components/base/modal/assistant-debugger-edit-section-modal/configure-audio-input-form';
import { ConfigureAudioOutputModalForm } from '@/app/components/base/modal/assistant-debugger-edit-section-modal/configure-audio-output-form';
import { TelephonyProvider } from '@/app/components/providers/telephony';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  MenuButton,
  MenuItem,
  MenuItemDivider,
  Checkbox,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  TableBatchActions,
  TableBatchAction,
  RadioButton,
  Tag,
} from '@carbon/react';
import { CornerBorderOverlay } from '@/app/components/base/corner-border';
import { useDeploymentSectionEdit } from './hooks/use-deployment-section-edit';

const DEPLOYMENT_LABELS = {
  debugger: 'Debugger',
  api: 'SDK / API',
  web: 'Web Widget',
  phone: 'Phone Call',
} as const;

type DeploymentSection =
  | 'telephony'
  | 'experience'
  | 'voice-input'
  | 'voice-output';
type DeploymentType = 'debugger' | 'api' | 'web' | 'phone';

const getSectionIcon = (section: DeploymentSection) => {
  if (section === 'experience') return Settings;
  if (section === 'voice-input') return Microphone;
  if (section === 'voice-output') return VolumeUp;
  return Phone;
};

export const ConfigureAssistantDeploymentPage = () => {
  const { assistantId } = useParams();
  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const navi = useGlobalNavigation();
  const { token, authId, projectId } = useCurrentCredential();
  const { showLoader, hideLoader } = useRapidaStore();

  const [isExpanded, setIsExpanded] = useState(false);
  const [isApiExpanded, setIsApiExpanded] = useState(false);
  const [isPhoneExpanded, setIsPhoneExpanded] = useState(false);
  const [isWidgetExpanded, setIsWidgetExpanded] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [copiedVersion, setCopiedVersion] = useState<string | null>(null);
  const [selectedDeploymentType, setSelectedDeploymentType] =
    useState<DeploymentType | null>(null);

  const get = useCallback(
    (id: typeof assistantId) => {
      if (id) {
        showLoader('block');
        const request = new GetAssistantRequest();
        const assistantDef = new AssistantDefinition();
        assistantDef.setAssistantid(id);
        request.setAssistantdefinition(assistantDef);
        GetAssistant(
          connectionConfig,
          request,
          ConnectionConfig.WithDebugger({
            authorization: token,
            userId: authId,
            projectId: projectId,
          }),
        )
          .then(epmr => {
            hideLoader();
            if (epmr?.getSuccess()) {
              const a = epmr.getData();
              if (a) setAssistant(a);
            } else {
              const error = epmr?.getError();
              toast.error(
                error?.getHumanmessage() ??
                  'Unable to get your assistant. please try again later.',
              );
            }
          })
          .catch(() => hideLoader());
      }
    },
    [token, authId, projectId],
  );

  useEffect(() => {
    get(assistantId);
  }, [assistantId]);

  const sectionEdit = useDeploymentSectionEdit(assistantId, () =>
    get(assistantId),
  );

  const deploymentCount = [
    assistant?.getApideployment(),
    assistant?.getWebplugindeployment(),
    assistant?.getDebuggerdeployment(),
    assistant?.getPhonedeployment(),
  ].filter(Boolean).length;

  const hasAnyDeployment = deploymentCount > 0;

  const rows: Array<{
    type: DeploymentType;
    source: string;
    name: string;
    version: string;
    sttProvider: string;
    ttsProvider: string;
    updated: string;
    onEdit: () => void;
    onDetails: () => void;
    onPreview?: () => void;
    sections: Array<{ label: string; value: DeploymentSection }>;
  }> = [];

  if (assistant?.hasDebuggerdeployment()) {
    const deployment = assistant.getDebuggerdeployment()!;
    rows.push({
      type: 'debugger',
      source: 'debugger',
      name: 'Debugger',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () => sectionEdit.openEditModal('debugger', 'experience'),
      onDetails: () => setIsExpanded(true),
      onPreview: () => navi.goToAssistantPreview(assistantId!),
      sections: [
        { label: 'Experience', value: 'experience' },
        { label: 'Voice Input', value: 'voice-input' },
        { label: 'Voice Output', value: 'voice-output' },
      ],
    });
  }

  if (assistant?.hasApideployment()) {
    const deployment = assistant.getApideployment()!;
    rows.push({
      type: 'api',
      source: 'sdk',
      name: 'SDK / API',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () => navi.goToConfigureApi(assistantId!),
      onDetails: () => setIsApiExpanded(true),
      sections: [
        { label: 'Experience', value: 'experience' },
        { label: 'Voice Input', value: 'voice-input' },
        { label: 'Voice Output', value: 'voice-output' },
      ],
    });
  }

  if (assistant?.hasPhonedeployment()) {
    const deployment = assistant.getPhonedeployment()!;
    rows.push({
      type: 'phone',
      source: 'phone-call',
      name: 'Phone Call',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () => navi.goToConfigureCall(assistantId!),
      onDetails: () => setIsPhoneExpanded(true),
      onPreview: () => navi.goToAssistantPreviewCall(assistantId!),
      sections: [
        { label: 'Telephony', value: 'telephony' },
        { label: 'Experience', value: 'experience' },
        { label: 'Voice Input', value: 'voice-input' },
        { label: 'Voice Output', value: 'voice-output' },
      ],
    });
  }

  if (assistant?.hasWebplugindeployment()) {
    const deployment = assistant.getWebplugindeployment()!;
    rows.push({
      type: 'web',
      source: 'web-plugin',
      name: 'Web Widget',
      version: deployment.getId() ? `vrsn_${deployment.getId()}` : '—',
      sttProvider: deployment.getInputaudio()?.getAudioprovider() || '—',
      ttsProvider: deployment.getOutputaudio()?.getAudioprovider() || '—',
      updated: deployment.getCreateddate()
        ? toHumanReadableDateTime(deployment.getCreateddate()!)
        : '—',
      onEdit: () => navi.goToConfigureWeb(assistantId!),
      onDetails: () => setIsWidgetExpanded(true),
      sections: [
        { label: 'Experience', value: 'experience' },
        { label: 'Voice Input', value: 'voice-input' },
        { label: 'Voice Output', value: 'voice-output' },
      ],
    });
  }

  const copyVersion = useCallback((version: string) => {
    navigator.clipboard.writeText(version);
    setCopiedVersion(version);
    setTimeout(() => setCopiedVersion(null), 2000);
  }, []);

  const addDeploymentMenu = useMemo(
    () => (
      <MenuButton
        className="deployment-add-menu"
        label="Add deployment"
        size="md"
        kind="primary"
        menuBorder
        menuBackgroundToken="background"
      >
        <MenuItem
          label="Debugger"
          onClick={() => navi.goToConfigureDebugger(assistantId!)}
        />
        <MenuItemDivider />
        <MenuItem
          label="Web Widget"
          onClick={() => navi.goToConfigureWeb(assistantId!)}
        />
        <MenuItemDivider />
        <MenuItem
          label="SDK / API"
          onClick={() => navi.goToConfigureApi(assistantId!)}
        />
        <MenuItemDivider />
        <MenuItem
          label="Phone Call"
          onClick={() => navi.goToConfigureCall(assistantId!)}
        />
      </MenuButton>
    ),
    [assistantId, navi],
  );

  const normalizedSearch = searchTerm.trim().toLowerCase();
  const filteredRows = normalizedSearch
    ? rows.filter(row =>
        [
          row.name,
          row.version,
          row.sttProvider,
          row.ttsProvider,
          row.updated,
          ...row.sections.map(s => s.label),
        ]
          .join(' ')
          .toLowerCase()
          .includes(normalizedSearch),
      )
    : rows;

  const selectedDeployment = filteredRows.find(
    row => row.type === selectedDeploymentType,
  );

  return (
    <div className="flex flex-col w-full flex-1 overflow-auto">
      {/* Modals */}
      {assistant?.getPhonedeployment() && (
        <AssistantPhoneCallDeploymentDialog
          modalOpen={isPhoneExpanded}
          setModalOpen={setIsPhoneExpanded}
          deployment={assistant.getPhonedeployment()!}
        />
      )}
      {assistant?.getDebuggerdeployment() && (
        <AssistantDebugDeploymentDialog
          modalOpen={isExpanded}
          setModalOpen={setIsExpanded}
          deployment={assistant.getDebuggerdeployment()!}
        />
      )}
      {assistant?.getWebplugindeployment() && (
        <AssistantWebWidgetlDeploymentDialog
          modalOpen={isWidgetExpanded}
          setModalOpen={setIsWidgetExpanded}
          deployment={assistant.getWebplugindeployment()!}
        />
      )}
      {assistant?.getApideployment() && (
        <AssistantApiDeploymentDialog
          modalOpen={isApiExpanded}
          setModalOpen={setIsApiExpanded}
          deployment={assistant.getApideployment()!}
        />
      )}
      {sectionEdit.activeEdit && (
        <DeploymentEditSectionModal
          modalOpen={!!sectionEdit.activeEdit}
          setModalOpen={isOpen => {
            if (!isOpen) sectionEdit.closeEditModal();
          }}
          section={sectionEdit.activeEdit.section}
          size={
            sectionEdit.activeEdit.section === 'experience' ||
            sectionEdit.activeEdit.section === 'telephony'
              ? 'md'
              : 'lg'
          }
          label={DEPLOYMENT_LABELS[sectionEdit.activeEdit.type]}
          errorMessage={sectionEdit.editError}
          isSaving={sectionEdit.isSaving}
          onSave={sectionEdit.saveSection}
        >
          {sectionEdit.activeEdit.section === 'telephony' && (
            <TelephonyProvider
              provider={sectionEdit.telephonyConfig.provider}
              parameters={sectionEdit.telephonyConfig.parameters}
              onChangeProvider={provider =>
                sectionEdit.setTelephonyConfig({ provider, parameters: [] })
              }
              onChangeParameter={parameters =>
                sectionEdit.setTelephonyConfig(c => ({ ...c, parameters }))
              }
            />
          )}
          {sectionEdit.activeEdit.section === 'experience' &&
            sectionEdit.activeEdit.type === 'web' && (
              <ConfigureWebExperienceModalForm
                experienceConfig={sectionEdit.experienceConfig}
                setExperienceConfig={sectionEdit.setExperienceConfig}
              />
            )}
          {sectionEdit.activeEdit.section === 'experience' &&
            sectionEdit.activeEdit.type !== 'web' && (
              <ConfigureExperienceModalForm
                experienceConfig={sectionEdit.experienceConfig}
                setExperienceConfig={sectionEdit.setExperienceConfig}
              />
            )}
          {sectionEdit.activeEdit.section === 'voice-input' && (
            <div className="space-y-4">
              <button
                type="button"
                onClick={() =>
                  sectionEdit.setVoiceInputEnable(!sectionEdit.voiceInputEnable)
                }
                className="relative group w-full text-left p-4 border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950/50 hover:bg-gray-50 dark:hover:bg-gray-900/60 transition-colors"
              >
                <CornerBorderOverlay
                  className={
                    sectionEdit.voiceInputEnable ? 'opacity-100' : undefined
                  }
                />
                <div onClick={e => e.stopPropagation()}>
                  <Checkbox
                    id="deployment-voice-input-toggle"
                    labelText="Enable voice input (Speech-to-Text)"
                    checked={sectionEdit.voiceInputEnable}
                    onChange={(_, { checked }) =>
                      sectionEdit.setVoiceInputEnable(checked)
                    }
                  />
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-6">
                  {sectionEdit.voiceInputEnable
                    ? 'Voice input is currently enabled.'
                    : 'Voice input is disabled. This deployment will not transcribe user speech.'}
                </p>
              </button>
              {sectionEdit.voiceInputEnable && (
                <ConfigureAudioInputModalForm
                  audioInputConfig={sectionEdit.audioInputConfig}
                  setAudioInputConfig={sectionEdit.setAudioInputConfig}
                />
              )}
            </div>
          )}
          {sectionEdit.activeEdit.section === 'voice-output' && (
            <div className="space-y-4">
              <button
                type="button"
                onClick={() =>
                  sectionEdit.setVoiceOutputEnable(
                    !sectionEdit.voiceOutputEnable,
                  )
                }
                className="relative group w-full text-left p-4 border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950/50 hover:bg-gray-50 dark:hover:bg-gray-900/60 transition-colors"
              >
                <CornerBorderOverlay
                  className={
                    sectionEdit.voiceOutputEnable ? 'opacity-100' : undefined
                  }
                />
                <div onClick={e => e.stopPropagation()}>
                  <Checkbox
                    id="deployment-voice-output-toggle"
                    labelText="Enable voice output (Text-to-Speech)"
                    checked={sectionEdit.voiceOutputEnable}
                    onChange={(_, { checked }) =>
                      sectionEdit.setVoiceOutputEnable(checked)
                    }
                  />
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-6">
                  {sectionEdit.voiceOutputEnable
                    ? 'Voice output is currently enabled.'
                    : 'Voice output is disabled. Assistant responses will be text only.'}
                </p>
              </button>
              {sectionEdit.voiceOutputEnable && (
                <ConfigureAudioOutputModalForm
                  audioOutputConfig={sectionEdit.audioOutputConfig}
                  setAudioOutputConfig={sectionEdit.setAudioOutputConfig}
                />
              )}
            </div>
          )}
        </DeploymentEditSectionModal>
      )}

      <Helmet title="Assistant deployment" />

      {/* Page header */}
      <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div>
          <Breadcrumb noTrailingSlash className="mb-2">
            <BreadcrumbItem
              href={`/deployment/assistant/${assistantId}/overview`}
            >
              Assistant
            </BreadcrumbItem>
          </Breadcrumb>
          <h1 className="text-2xl font-light tracking-tight">Deployments</h1>
        </div>
      </div>

      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={!!selectedDeployment}
          totalSelected={selectedDeployment ? 1 : 0}
          totalCount={filteredRows.length}
          onCancel={() => setSelectedDeploymentType(null)}
          className="[&_[class*=divider]]:hidden [&_.cds--btn]:transition-colors [&_.cds--btn:hover]:!bg-primary [&_.cds--btn:hover]:!text-white"
        >
          {selectedDeployment && (
            <>
              <TableBatchAction
                renderIcon={Edit}
                kind="ghost"
                onClick={() => {
                  selectedDeployment.onEdit();
                  setSelectedDeploymentType(null);
                }}
              >
                Edit deployment
              </TableBatchAction>
              {selectedDeployment.sections.map(item => (
                <TableBatchAction
                  key={`batch-${selectedDeployment.type}-${item.value}`}
                  renderIcon={getSectionIcon(item.value)}
                  kind="ghost"
                  onClick={() => {
                    sectionEdit.openEditModal(
                      selectedDeployment.type,
                      item.value,
                    );
                    setSelectedDeploymentType(null);
                  }}
                >
                  {`Edit ${item.label}`}
                </TableBatchAction>
              ))}
            </>
          )}
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search deployments"
            onChange={(e: any) => setSearchTerm(e.target?.value || '')}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={() => get(assistantId)}
          />
          {addDeploymentMenu}
        </TableToolbarContent>
      </TableToolbar>

      {hasAnyDeployment ? (
        <>
          {filteredRows.length > 0 ? (
            <div className="overflow-auto flex-1">
              <Table>
                <TableHead>
                  <TableRow>
                    <TableHeader className="!w-12" />
                    <TableHeader>Channel</TableHeader>
                    <TableHeader>Version</TableHeader>
                    <TableHeader>STT Provider</TableHeader>
                    <TableHeader>TTS Provider</TableHeader>
                    <TableHeader>Updated</TableHeader>
                    <TableHeader>Actions</TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredRows.map(row => (
                    <TableRow
                      key={row.type}
                      isSelected={selectedDeploymentType === row.type}
                      onClick={() =>
                        setSelectedDeploymentType(
                          selectedDeploymentType === row.type ? null : row.type,
                        )
                      }
                      className="cursor-pointer"
                    >
                      <TableCell
                        className="!w-12 !pr-0"
                        onClick={e => e.stopPropagation()}
                      >
                        <RadioButton
                          id={`deployment-select-${row.type}`}
                          name="deployment-select"
                          labelText=""
                          hideLabel
                          checked={selectedDeploymentType === row.type}
                          onChange={() =>
                            setSelectedDeploymentType(
                              selectedDeploymentType === row.type
                                ? null
                                : row.type,
                            )
                          }
                        />
                      </TableCell>
                      <TableCell>
                        <SourceIndicator source={row.source} />
                      </TableCell>
                      <TableCell className="!font-mono !text-xs">
                        {row.version !== '—' ? (
                          <span className="inline-flex items-center gap-1">
                            {row.version}
                            <Button
                              hasIconOnly
                              renderIcon={
                                copiedVersion === row.version ? Checkmark : Copy
                              }
                              iconDescription="Copy version id"
                              kind="ghost"
                              size="sm"
                              onClick={() => copyVersion(row.version)}
                              className="!min-h-0 !p-1"
                            />
                          </span>
                        ) : (
                          row.version
                        )}
                      </TableCell>
                      <TableCell>
                        <AudioProviderTag
                          provider={row.sttProvider}
                          icon={<Microphone size={14} />}
                        />
                      </TableCell>
                      <TableCell>
                        <AudioProviderTag
                          provider={row.ttsProvider}
                          icon={<VolumeUp size={14} />}
                        />
                      </TableCell>
                      <TableCell>{row.updated}</TableCell>
                      <TableCell onClick={e => e.stopPropagation()}>
                        <div className="flex items-center gap-0">
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={View}
                            iconDescription="View details"
                            onClick={row.onDetails}
                          />
                          <IconOnlyButton
                            kind="ghost"
                            size="md"
                            renderIcon={Edit}
                            iconDescription="Edit deployment"
                            onClick={row.onEdit}
                          />
                          {row.onPreview && (
                            <IconOnlyButton
                              kind="ghost"
                              size="md"
                              renderIcon={Launch}
                              iconDescription="Preview deployment"
                              onClick={row.onPreview}
                            />
                          )}
                          {row.sections.map(item => (
                            <IconOnlyButton
                              key={`${row.type}-${item.value}`}
                              kind="ghost"
                              size="md"
                              renderIcon={getSectionIcon(item.value)}
                              iconDescription={`Edit ${item.label}`}
                              onClick={() =>
                                sectionEdit.openEditModal(row.type, item.value)
                              }
                            />
                          ))}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <EmptyState
              icon={Deploy}
              title="No deployments found"
              subtitle="No deployment matched your search."
            />
          )}
        </>
      ) : (
        <EmptyState
          icon={Deploy}
          title="No deployments found"
          subtitle="Any assistant deployments you configure will be listed here."
        />
      )}
    </div>
  );
};

const providerLabels: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  google: 'Google',
  gemini: 'Gemini',
  azure: 'Azure',
  'azure-openai': 'Azure OpenAI',
  groq: 'Groq',
  mistral: 'Mistral',
  cohere: 'Cohere',
  deepseek: 'DeepSeek',
  deepgram: 'Deepgram',
  cartesia: 'Cartesia',
  elevenlabs: 'ElevenLabs',
};

function AudioProviderTag({
  provider,
  icon,
}: {
  provider?: string;
  icon: ReactNode;
}) {
  const key = provider?.toLowerCase() || '';
  const label =
    key && key !== '—'
      ? providerLabels[key] || provider || 'Unknown'
      : 'Unavailable';

  return (
    <Tag size="md" type="cool-gray">
      <span className="inline-flex items-center gap-1.5 leading-none">
        {icon}
        {label}
      </span>
    </Tag>
  );
}
