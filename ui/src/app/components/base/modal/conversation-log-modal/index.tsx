import { AssistantConversationMessage } from '@rapidaai/react';
import { Metadata } from '@rapidaai/react';
import { Tabs } from '@/app/components/carbon/tabs';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { CodeHighlighting } from '@/app/components/code-highlighting';
import { MarkdownViewer } from '@/app/components/markdown-viewer';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { Chat } from '@carbon/icons-react';
import { FC, useState } from 'react';

interface ConversationLogDialogProps extends ModalProps {
  currentAssistantMessage: AssistantConversationMessage;
}
/**
 *
 * @param props
 * @returns
 */
export function ConversationLogDialog(props: ConversationLogDialogProps) {
  const [selectedTab, setSelectedTab] = useState(0);

  return (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-[580px]"
      label="Conversation Log"
      title={props.currentAssistantMessage.getAssistantconversationid()}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        <Tabs
          tabs={['Message', 'Metrics', 'Metadata']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          aria-label="Conversation log tabs"
          className="!h-full !min-h-0 !flex !flex-col [&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-gray-200 dark:[&_.cds--tabs__nav]:border-gray-800 [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0"
          panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
        >
          <div className="h-full overflow-auto">
            {props.currentAssistantMessage.getBody() ? (
              <div className="border border-gray-200 dark:border-gray-800">
                <MarkdownViewer
                  text={props.currentAssistantMessage.getBody()}
                />
              </div>
            ) : (
              <div className="h-full flex items-center justify-center py-8">
                <EmptyState
                  icon={Chat}
                  title="No Message"
                  subtitle="Message will be available here after execution completes."
                />
              </div>
            )}
          </div>
          <div className="h-full min-h-0">
            <CodeHighlighting
              className="!h-full !min-h-0"
              lang="json"
              lineNumbers={false}
              foldGutter={false}
              code={JSON.stringify(
                props.currentAssistantMessage
                  ?.getMetricsList()
                  .map(metric => metric.toObject()),
                null,
                2,
              )}
            />
          </div>
          <MessageMetadatas
            metadata={props.currentAssistantMessage.getMetadataList()}
          />
        </Tabs>
      </div>
    </RightSideModal>
  );
}

const MessageMetadatas: FC<{ metadata: Array<Metadata> }> = ({ metadata }) => {
  if (metadata.length <= 0)
    return (
      <div className="h-full flex items-center justify-center py-8">
        <EmptyState
          icon={Chat}
          title="No Metadata"
          subtitle="There is no metadata for this message."
        />
      </div>
    );
  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
      {metadata.map((x, idx) => (
        <div
          key={`metadata-idx-${idx}`}
          className="flex items-center justify-between h-12 px-4 gap-4"
        >
          <span className="text-xs font-medium uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400 shrink-0">
            {x.getKey()}
          </span>
          <div className="flex items-center text-sm text-gray-900 dark:text-gray-100">
            {x.getValue()}
          </div>
        </div>
      ))}
    </div>
  );
};
