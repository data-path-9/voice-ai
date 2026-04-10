import React, { useState, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ConnectionConfig, ServiceError } from '@rapidaai/react';
import { Tabs } from '@/app/components/carbon/tabs';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { GetWebhookLog } from '@rapidaai/react';
import {
  AssistantWebhookLog,
  GetAssistantWebhookLogResponse,
} from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface WebhookLogModalProps extends ModalProps {
  currentWebhookId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function WebhookLogDialog(props: WebhookLogModalProps) {
  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [activity, setActivity] = useState<AssistantWebhookLog | null>(null);
  const [selectedTab, setSelectedTab] = useState(0);

  const getActivity = (currentProject: string, currentActivityId) => {
    return GetWebhookLog(
      connectionConfig,
      currentProject,
      currentActivityId,
      afterActivities,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: userId,
        projectId: projectId,
      }),
    );
  };

  /**
   *
   */
  useEffect(() => {
    showLoader('overlay');
    getActivity(projectId, props.currentWebhookId);
  }, [projectId, props.currentWebhookId]);

  /**
   *
   * @param err
   * @param at
   */
  const afterActivities = (
    err: ServiceError | null,
    at: GetAssistantWebhookLogResponse | null,
  ) => {
    hideLoader();
    if (at?.getSuccess()) {
      let data = at.getData();
      if (data) {
        setActivity(data);
      }
    } else {
      let err = at?.getError();
      if (err) toast.error(err?.getHumanmessage());
      toast.error('Unable to resolve the request, please try again later.');
    }
  };

  return (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-[580px]"
      label="Webhook Log"
      title={props.currentWebhookId}
    >
      <div className="relative flex-1 flex flex-col min-h-0">
        <Tabs
          tabs={['Request', 'Response']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          aria-label="Webhook log tabs"
          className="!h-full !min-h-0 !flex !flex-col [&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-gray-200 dark:[&_.cds--tabs__nav]:border-gray-800 [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0"
          panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
        >
          <div className="h-full min-h-0">
            <CodeHighlighting
              className="!h-full !min-h-0"
              code={JSON.stringify(
                activity?.getRequest()?.toJavaScript(),
                null,
                2,
              )}
            />
          </div>
          <div className="h-full min-h-0">
            <CodeHighlighting
              className="!h-full !min-h-0"
              code={JSON.stringify(
                activity?.getResponse()?.toJavaScript(),
                null,
                2,
              )}
            />
          </div>
        </Tabs>
      </div>
    </RightSideModal>
  );
}
