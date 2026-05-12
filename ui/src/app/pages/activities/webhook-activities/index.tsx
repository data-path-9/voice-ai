import { useState, useEffect } from 'react';
import { Helmet } from '@/app/components/helmet';
import { DateFilter } from '@/app/components/carbon/date-filter';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import {
  formatNanoToReadableMilli,
  toDateString,
  toHumanReadableDateTime,
} from '@/utils/date';
import { HttpStatusSpanIndicator } from '@/app/components/indicators/http-status';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { useWebhookLogPage } from '@/hooks/use-webhook-log-page-store';
import { RequestLogDialog } from '@/app/components/base/modal/webhook-log-modal';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import {
  RetryAssistantHTTPLogRequest,
  RetryHTTPLog,
} from '@rapidaai/react';
import { connectionConfig } from '@/configs';

import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Loading,
  Tag,
  Link,
} from '@carbon/react';
import { Pagination } from '@/app/components/carbon/pagination';
import { IconOnlyButton } from '@/app/components/carbon/button';
import { Renew, View, EventSchedule, Launch } from '@carbon/icons-react';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { ScrollableTableSection } from '@/app/components/sections/table-section';

export function ListingPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token, projectId] = useCredential();
  const [currentActivityId, setCurrentActivityId] = useState('');
  const [showLogModal, setShowLogModal] = useState(false);
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Retry request?',
    content:
      'This will re-run the selected HTTP request. Do you want to continue?',
  });

  const {
    getActivities,
    addCriterias,
    webhookLogs,
    onChangeActivities,
    columns,
    page,
    setPage,
    totalCount,
    criteria,
    pageSize,
    visibleColumn,
    setPageSize,
    setColumns,
  } = useWebhookLogPage();

  const onDateSelect = (to: Date, from: Date) => {
    addCriterias([
      { k: 'created_date', v: toDateString(to), logic: '<=' },
      { k: 'created_date', v: toDateString(from), logic: '>=' },
    ]);
  };

  useEffect(() => {
    showLoader();
    onGetActivities();
  }, [projectId, page, pageSize, JSON.stringify(criteria)]);

  const onGetActivities = () => {
    getActivities(
      projectId,
      token,
      userId,
      err => {
        hideLoader();
        toast.error(err);
      },
      logs => {
        hideLoader();
        onChangeActivities(logs);
      },
    );
  };

  const retryRequestLog = async (requestLogId: string) => {
    showLoader();
    const request = new RetryAssistantHTTPLogRequest();
    request.setProjectid(projectId);
    request.setId(requestLogId);

    try {
      const response = await RetryHTTPLog(connectionConfig, request, {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      });

      if (response?.getSuccess()) {
        toast.success('Request retried successfully.');
        onGetActivities();
        return;
      }

      const message = response?.getError()?.getHumanmessage();
      toast.error(message || 'Unable to retry the request, please try again.');
    } catch {
      toast.error('Unable to retry the request, please try again.');
    } finally {
      hideLoader();
    }
  };

  const visibleColumns = columns.filter(c => c.visible);

  return (
    <>
      <ConfirmDialogComponent />
      {currentActivityId && (
        <RequestLogDialog
          modalOpen={showLogModal}
          setModalOpen={setShowLogModal}
          currentRequestLogId={currentActivityId}
        />
      )}

      <div className="h-full flex flex-col overflow-hidden">
        <Helmet title="Request Logs" />
        <PageHeaderBlock>
          <PageTitleWithCount count={webhookLogs.length} total={totalCount}>
            Request Logs
          </PageTitleWithCount>
        </PageHeaderBlock>

        <TableToolbar>
          <TableToolbarContent>
            <TableToolbarSearch placeholder="Search request logs" />
            <DateFilter
              onApply={(from, to) => onDateSelect(to, from)}
              onReset={() => addCriterias([])}
            />
            <IconOnlyButton
              kind="ghost"
              size="lg"
              renderIcon={Renew}
              iconDescription="Refresh"
              onClick={() => onGetActivities()}
            />
          </TableToolbarContent>
        </TableToolbar>

        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loading withOverlay={false} small />
          </div>
        ) : webhookLogs.length > 0 ? (
          <ScrollableTableSection>
            <Table className="min-w-max">
            <TableHead>
              <TableRow>
                {visibleColumns.map(col => (
                  <TableHeader key={col.key}>{col.name}</TableHeader>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {webhookLogs.map((at, idx) => (
                <TableRow key={idx}>
                  {visibleColumn('sourcerefid') && (
                  <TableCell className="text-[13px]">
                      <span className="font-mono">
                        {at.getSourcerefid()}
                      </span>
                  </TableCell>
                  )}
                  {visibleColumn('sessionid') && (
                    <TableCell className="text-sm">
                      <Link
                        href={`/deployment/assistant/${at.getAssistantid()}/sessions/${at.getAssistantconversationid()}`}
                        className="!text-sm !inline-flex !items-center !gap-1"
                      >
                        <span>{at.getAssistantconversationid()}</span>
                        <Launch size={12} />
                      </Link>
                    </TableCell>
                  )}
                  {visibleColumn('event') && (
                    <TableCell className="text-sm">
                      <Tag size="sm" type="blue">
                        {at.getSourceevent()}
                      </Tag>
                    </TableCell>
                  )}
                  {visibleColumn('endpoint') && (
                    <TableCell className="text-sm">
                      {at.getHttpmethod()}:{at.getHttpurl()}
                    </TableCell>
                  )}
                  {visibleColumn('action') && (
                    <TableCell className="text-sm">
                      <IconOnlyButton
                        kind="ghost"
                        size="md"
                        renderIcon={Renew}
                        iconDescription="Retry request"
                        onClick={() =>
                          showDialog(() => retryRequestLog(at.getId()))
                        }
                      />
                      <IconOnlyButton
                        kind="ghost"
                        size="md"
                        renderIcon={View}
                        iconDescription="View detail"
                        onClick={() => {
                          setCurrentActivityId(at.getId());
                          setShowLogModal(true);
                        }}
                      />
                    </TableCell>
                  )}
                  {visibleColumn('responsestatus') && (
                    <TableCell className="text-sm">
                      <HttpStatusSpanIndicator
                        status={Number(at.getResponsestatus())}
                      />
                    </TableCell>
                  )}
                  {visibleColumn('timetaken') && (
                    <TableCell className="font-mono text-[13px]">
                      {formatNanoToReadableMilli(at.getTimetaken())}
                    </TableCell>
                  )}
                  {visibleColumn('retrycount') && (
                    <TableCell className="text-sm">
                      {at.getRetrycount()}
                    </TableCell>
                  )}
                  {visibleColumn('created_date') && (
                    <TableCell className="text-[13px] whitespace-nowrap">
                      {at.getCreateddate() &&
                        toHumanReadableDateTime(at.getCreateddate()!)}
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
            </Table>
          </ScrollableTableSection>
        ) : (
          <EmptyState
            icon={EventSchedule}
            title="No request logs found"
            subtitle="HTTP request logs will appear here once requests are triggered by assistant workflows."
          />
        )}

        {webhookLogs.length > 0 && (
          <Pagination
            totalItems={totalCount}
            page={page}
            pageSize={pageSize}
            pageSizes={[10, 20, 25, 50, 100]}
            onChange={({ page: p, pageSize: ps }) => {
              if (ps !== pageSize) setPageSize(ps);
              else setPage(p);
            }}
          />
        )}
      </div>
    </>
  );
}
