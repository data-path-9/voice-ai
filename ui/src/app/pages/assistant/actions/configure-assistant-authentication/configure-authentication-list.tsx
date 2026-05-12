import { FC, useEffect, useMemo, useState } from 'react';
import {
  AssistantAuthentication,
  DisableAssistantAuthentication,
  DisableAssistantAuthenticationRequest,
  GetAssistantAuthentication,
  GetAssistantAuthenticationRequest,
} from '@rapidaai/react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
} from '@carbon/react';
import { Renew, Add, Edit, DisableStep } from '@carbon/icons-react';
import toast from 'react-hot-toast/headless';

import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { connectionConfig } from '@/configs';
import { PrimaryButton, IconOnlyButton } from '@/app/components/carbon/button';
import { SectionLoader } from '@/app/components/loader/section-loader';
import { TableSection } from '@/app/components/sections/table-section';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { CarbonShapeIndicator } from '@/app/components/carbon/shape-indicator';
import { toHumanReadableDateTime } from '@/utils/date';

import {
  getAuthenticationStatus,
  toOptionMap,
  AUTH_OPTION_ENDPOINT,
  AUTH_OPTION_METHOD,
} from './shared';

interface ConfigureAuthenticationListProps {
  assistantId: string;
}

export const ConfigureAuthenticationList: FC<
  ConfigureAuthenticationListProps
> = ({ assistantId }) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Disable authentication?',
    content: 'Authentication will be inactive for this assistant.',
  });

  const [loading, setLoading] = useState(true);
  const [authentication, setAuthentication] =
    useState<AssistantAuthentication | null>(null);

  const configureRoute = `/deployment/assistant/${assistantId}/configure-authentication/${
    authentication ? 'edit' : 'create'
  }`;

  const load = () => {
    setLoading(true);
    const request = new GetAssistantAuthenticationRequest();
    request.setAssistantid(assistantId);

    GetAssistantAuthentication(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        if (!response?.getSuccess()) {
          setAuthentication(null);
          setLoading(false);
          return;
        }

        setAuthentication(response.getData() || null);
        setLoading(false);
      })
      .catch(() => {
        setAuthentication(null);
        setLoading(false);
      });
  };

  useEffect(() => {
    load();
  }, [assistantId, authId, token, projectId]);

  const optionMap = useMemo(
    () => toOptionMap(authentication?.getOptionsList?.() || []),
    [authentication],
  );

  const onDisable = () => {
    if (!authentication) return;
    const request = new DisableAssistantAuthenticationRequest();
    request.setAssistantid(assistantId);
    DisableAssistantAuthentication(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        if (response?.getSuccess()) {
          toast.success('Assistant authentication disabled successfully.');
          load();
          return;
        }
        toast.error(
          response?.getError?.()?.getHumanmessage?.() ||
            'Unable to disable assistant authentication.',
        );
      })
      .catch(err => {
        toast.error(
          err?.message || 'Unable to disable assistant authentication.',
        );
      });
  };

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col flex-1">
      <ConfirmDialogComponent />
      <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div>
          <Breadcrumb noTrailingSlash className="mb-2">
            <BreadcrumbItem
              href={`/deployment/assistant/${assistantId}/overview`}
            >
              Assistant
            </BreadcrumbItem>
          </Breadcrumb>
          <h1 className="text-2xl font-light tracking-tight">Authentication</h1>
        </div>
      </div>

      <TableToolbar>
        <TableToolbarContent>
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={load}
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            onClick={() => navigator.goTo(configureRoute)}
          >
            Configure authentication
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>

      <TableSection>
        {authentication ? (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader>Provider Type</TableHeader>
                <TableHeader>Method</TableHeader>
                <TableHeader>URL</TableHeader>
                <TableHeader>Status</TableHeader>
                <TableHeader>Date</TableHeader>
                <TableHeader>Actions</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell className="text-sm whitespace-nowrap">
                  HTTP
                </TableCell>
                <TableCell className="text-sm whitespace-nowrap">
                  {optionMap[AUTH_OPTION_METHOD] || '-'}
                </TableCell>
                <TableCell className="text-sm max-w-[360px] truncate">
                  {optionMap[AUTH_OPTION_ENDPOINT] || '-'}
                </TableCell>
                <TableCell className="text-sm whitespace-nowrap">
                  <CarbonShapeIndicator
                    state={authentication.getStatus()}
                    textSize={14}
                  />
                </TableCell>
                <TableCell className="text-[13px] whitespace-nowrap">
                  {authentication.getCreateddate() &&
                    toHumanReadableDateTime(authentication.getCreateddate()!)}
                </TableCell>
                <TableCell
                  className="text-sm whitespace-nowrap"
                  onClick={e => e.stopPropagation()}
                >
                  <div className="flex items-center gap-0">
                    <IconOnlyButton
                      kind="ghost"
                      size="md"
                      renderIcon={Edit}
                      iconDescription="Configure authentication"
                      onClick={() => navigator.goTo(configureRoute)}
                    />
                    <IconOnlyButton
                      size="md"
                      kind="ghost"
                      renderIcon={DisableStep}
                      iconDescription="Disable authentication"
                      disabled={
                        getAuthenticationStatus(authentication) !== 'active'
                      }
                      onClick={() => showDialog(onDisable)}
                    />
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        ) : (
          <EmptyState
            icon={Add}
            title="No authentication configured"
            subtitle="Create authentication to verify sessions before initialization."
          />
        )}
      </TableSection>
    </div>
  );
};
