import React, { useState, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { useCredential } from '@/hooks/use-credential';
import { OverviewRow } from '@/app/components/base/modal/overview-row';

import {
  AuditLog,
  GetAuditLogResponse,
  GetActivity,
  ConnectionConfig,
} from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { Metadata } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { Tabs } from '@/app/components/carbon/tabs';
import { CarbonStatusIndicator } from '@/app/components/carbon/status-indicator';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { HttpStatusSpanIndicator } from '@/app/components/indicators/http-status';
import { connectionConfig } from '@/configs';
import { CodeHighlighting } from '@/app/components/code-highlighting';
import { toHumanReadableDateTime } from '@/utils/date';

interface LLMLogModalProps extends ModalProps {
  currentActivityId: string;
}
/**
 *
 * @param props
 * @returns
 */
export function LLMLogDialog(props: LLMLogModalProps) {
  /**
   * user credentials
   */
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  /**
   *
   */
  const [additionalData, setAdditionalData] = useState<Metadata[]>([]);
  /**
   *
   */
  const [activity, setActivity] = useState<AuditLog | null>(null);
  const [selectedTab, setSelectedTab] = useState(0);

  const getActivity = (currentProject: string, currentActivityId) => {
    return GetActivity(
      connectionConfig,
      currentProject,
      currentActivityId,
      afterActivities,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
    );
  };

  /**
   *
   */
  useEffect(() => {
    showLoader('overlay');
    getActivity(projectId, props.currentActivityId);
  }, [projectId, props.currentActivityId]);

  /**
   *
   * @param err
   * @param at
   */
  const afterActivities = (
    err: ServiceError | null,
    at: GetAuditLogResponse | null,
  ) => {
    hideLoader();
    if (at?.getSuccess()) {
      let data = at.getData();
      if (data) {
        setActivity(data);
        setAdditionalData(data?.getExternalauditmetadatasList());
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
      label="LLM Log"
      title={props.currentActivityId}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        <Tabs
          tabs={['Overview', 'Request', 'Response', 'Metrics']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          aria-label="LLM log tabs"
          className="!h-full !min-h-0 !flex !flex-col [&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-gray-200 dark:[&_.cds--tabs__nav]:border-gray-800 [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0"
          panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
        >
          <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
            {activity && (
              <>
                <OverviewRow label="Status">
                  <CarbonStatusIndicator state={activity.getStatus()} />
                </OverviewRow>
                <OverviewRow label="Time Taken">
                  <span className="text-sm tabular-nums text-gray-900 dark:text-gray-100">
                    {`${activity.getTimetaken() / 1000000}ms`}
                  </span>
                </OverviewRow>
                <OverviewRow label="Created">
                  <span className="text-sm text-gray-900 dark:text-gray-100">
                    {toHumanReadableDateTime(activity.getCreateddate()!)}
                  </span>
                </OverviewRow>
                {activity?.getResponsestatus() && (
                  <OverviewRow label="Response Status">
                    <HttpStatusSpanIndicator
                      status={activity.getResponsestatus()}
                    />
                  </OverviewRow>
                )}
                {additionalData.map((ad, idx) => (
                  <OverviewRow
                    key={idx}
                    label={ad.getKey().replaceAll('_', ' ')}
                  >
                    <span className="text-sm text-gray-900 dark:text-gray-100 truncate max-w-[20rem]">
                      {ad.getValue()}
                    </span>
                  </OverviewRow>
                ))}
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
          <div className="h-full min-h-0">
            <CodeHighlighting
              className="!h-full !min-h-0"
              lang="json"
              lineNumbers={false}
              foldGutter={false}
              code={JSON.stringify(
                activity?.getMetricsList().map(metric => metric.toObject()),
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
