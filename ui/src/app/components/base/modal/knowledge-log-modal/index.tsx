import React, { useState, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';
import { OverviewRow } from '@/app/components/base/modal/overview-row';
import {
  ConnectionConfig,
  GetKnowledgeLog,
  GetKnowledgeLogRequest,
  KnowledgeLog,
} from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { Tabs } from '@/app/components/carbon/tabs';
import { StatusIndicator } from '@/app/components/indicators/status';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';
import { toHumanReadableDateTime } from '@/utils/date';

interface KnowledgeLogModalProps extends ModalProps {
  currentActivityId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function KnowledgeLogDialog(props: KnowledgeLogModalProps) {
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const [activity, setActivity] = useState<KnowledgeLog | null>(null);
  const [selectedTab, setSelectedTab] = useState(0);

  const getActivity = (currentProject: string, currentActivityId) => {
    const request = new GetKnowledgeLogRequest();
    request.setId(currentActivityId);
    request.setProjectid(currentProject);
    return GetKnowledgeLog(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
    )
      .then(at => {
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
      })
      .catch(x => {
        hideLoader();
        toast.error('Unable to resolve the request, please try again later.');
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
      label="Knowledge Log"
      title={props.currentActivityId}
    >
      <div className="relative flex-1 flex flex-col min-h-0">
        <Tabs
          tabs={['Overview', 'Request', 'Response']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          aria-label="Knowledge log tabs"
          className="!h-full !min-h-0 !flex !flex-col [&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-gray-200 dark:[&_.cds--tabs__nav]:border-gray-800 [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0"
          panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
        >
          <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
            {activity && (
              <>
                <OverviewRow label="Status">
                  <StatusIndicator state={activity.getStatus()} size="small" />
                </OverviewRow>
                <OverviewRow label="Time Taken">
                  <span className="text-sm tabular-nums text-gray-900 dark:text-gray-100">
                    {`${Number(activity.getTimetaken()) / 1000000}ms`}
                  </span>
                </OverviewRow>
                <OverviewRow label="Created">
                  <span className="text-sm text-gray-900 dark:text-gray-100">
                    {toHumanReadableDateTime(activity.getCreateddate()!)}
                  </span>
                </OverviewRow>
              </>
            )}
          </div>
          <div className="h-full min-h-0">
            <CodeHighlighting
              className="!h-full !min-h-0"
              lang="json"
              lineNumbers={false}
              foldGutter={false}
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
              lang="json"
              lineNumbers={false}
              foldGutter={false}
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
