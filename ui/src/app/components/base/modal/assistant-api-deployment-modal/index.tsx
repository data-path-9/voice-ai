import {
  AssistantApiDeployment,
  DeploymentAudioProvider,
} from '@rapidaai/react';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { CopyButton } from '@/app/components/carbon/button/copy-button';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { ProviderPill } from '@/app/components/pill/provider-model-pill';
import { FC, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  DeploymentRow,
  DeploymentSectionHeader,
} from '@/app/components/base/modal/deployment-modal-primitives';
import { Tabs } from '@/app/components/carbon/tabs';

interface AssistantApiDeploymentDialogProps extends ModalProps {
  deployment: AssistantApiDeployment;
}

export function AssistantApiDeploymentDialog(
  props: AssistantApiDeploymentDialogProps,
) {
  const [selectedTab, setSelectedTab] = useState(0);
  const modalContent = (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-[580px]"
      label="SDK / API Deployment"
      title={props.deployment.getId()}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        <Tabs
          tabs={['Audio']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          aria-label="SDK/API deployment tabs"
          className="!h-full !min-h-0 !flex !flex-col [&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-gray-200 dark:[&_.cds--tabs__nav]:border-gray-800 [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0"
          panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
        >
          <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
            <VoiceInput deployment={props.deployment?.getInputaudio()} />
            <VoiceOutput deployment={props.deployment?.getOutputaudio()} />
          </div>
        </Tabs>
      </div>
    </RightSideModal>
  );

  if (typeof document === 'undefined') return modalContent;

  return createPortal(modalContent, document.body);
}

const Row = DeploymentRow;
const SectionHeader = DeploymentSectionHeader;

const VoiceInput: FC<{ deployment?: DeploymentAudioProvider }> = ({
  deployment,
}) => (
  <>
    <SectionHeader label="Speech to text" />
    {deployment?.getAudiooptionsList() ? (
      deployment?.getAudiooptionsList().length > 0 && (
        <>
          <Row label="Provider">
            <ProviderPill provider={deployment?.getAudioprovider()} />
          </Row>
          {deployment
            ?.getAudiooptionsList()
            .filter(d => d.getValue())
            .filter(d => d.getKey().startsWith('listen.'))
            .map((detail, index) => (
              <Row key={index} label={detail.getKey()}>
                <span className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate max-w-[200px] text-right">
                  {detail.getValue()}
                </span>
                <CopyButton className="h-6 w-6 shrink-0">
                  {detail.getValue()}
                </CopyButton>
              </Row>
            ))}
        </>
      )
    ) : (
      <div className="px-4 py-3">
        <YellowNoticeBlock>Voice input is not enabled</YellowNoticeBlock>
      </div>
    )}
  </>
);

const VoiceOutput: FC<{ deployment?: DeploymentAudioProvider }> = ({
  deployment,
}) => (
  <>
    <SectionHeader label="Text to speech" />
    {deployment?.getAudiooptionsList() ? (
      deployment?.getAudiooptionsList().length > 0 && (
        <>
          <Row label="Provider">
            <ProviderPill provider={deployment?.getAudioprovider()} />
          </Row>
          {deployment
            ?.getAudiooptionsList()
            .filter(d => d.getValue())
            .filter(d => d.getKey().startsWith('speak.'))
            .map((detail, index) => (
              <Row key={index} label={detail.getKey()}>
                <span className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate max-w-[200px] text-right">
                  {detail.getValue()}
                </span>
                <CopyButton className="h-6 w-6 shrink-0">
                  {detail.getValue()}
                </CopyButton>
              </Row>
            ))}
        </>
      )
    ) : (
      <div className="px-4 py-3">
        <YellowNoticeBlock>Voice output is not enabled</YellowNoticeBlock>
      </div>
    )}
  </>
);
