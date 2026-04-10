import React, { useState, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';

import {
  ConnectionConfig,
  GetAssistantToolLog,
  GetAssistantToolLogRequest,
  AssistantToolLog,
} from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { Tabs } from '@/app/components/carbon/tabs';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';

interface ToolLogModalProps extends ModalProps {
  currentActivityId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function ToolLogDialog(props: ToolLogModalProps) {
  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [activity, setActivity] = useState<AssistantToolLog | null>(null);
  const [selectedTab, setSelectedTab] = useState(0);
  const getActivity = (currentProject: string, currentActivityId) => {
    const request = new GetAssistantToolLogRequest();
    request.setProjectid(currentProject);
    request.setId(currentActivityId);
    GetAssistantToolLog(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
    ).then(at => {
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
    });
  };

  /**
   *
   */
  useEffect(() => {
    showLoader('overlay');
    getActivity(projectId, props.currentActivityId);
  }, [projectId, props.currentActivityId]);

  return (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-[580px]"
      label="Tool Log"
      title={props.currentActivityId}
    >
      <div className="relative flex-1 flex flex-col min-h-0">
        <Tabs
          tabs={['Request', 'Response']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          aria-label="Tool log tabs"
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
