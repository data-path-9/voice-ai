import React, { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { toHumanReadableDateTime } from '@/utils/date';
import { Activity, Edit, TrashCan, Add, Renew } from '@carbon/icons-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { SectionLoader } from '@/app/components/loader/section-loader';
import toast from 'react-hot-toast/headless';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { CreateAssistantTelemetry } from './create-assistant-telemetry';
import { UpdateAssistantTelemetry } from './update-assistant-telemetry';
import { useAssistantTelemetryPageStore } from '@/app/pages/assistant/actions/store/use-telemetry-page-store';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { TELEMETRY_PROVIDER } from '@/providers';
import { IconOnlyButton, PrimaryButton } from '@/app/components/carbon/button';
import { Tag } from '@carbon/react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
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
} from '@carbon/react';

export function ConfigureAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantTelemetry assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAssistantTelemetry assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantTelemetryPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <UpdateAssistantTelemetry assistantId={assistantId} />}</>
  );
}

const providerNameByCode = new Map(
  TELEMETRY_PROVIDER.map(p => [p.code, p.name]),
);

const ConfigureAssistantTelemetry: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigation = useGlobalNavigation();
  const action = useAssistantTelemetryPageStore();
  const { authId, token, projectId } = useCurrentCredential();
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedTelemetryId, setSelectedTelemetryId] = useState<string | null>(
    null,
  );
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({
    title: 'Delete telemetry?',
    content: 'This telemetry provider will be removed from the assistant.',
  });

  useEffect(() => {
    get();
  }, []);

  const get = () => {
    setLoading(true);
    action.getAssistantTelemetry(
      assistantId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setLoading(false);
      },
      () => {
        setLoading(false);
      },
    );
  };

  const deleteTelemetry = (telemetryId: string) => {
    setLoading(true);
    action.deleteAssistantTelemetry(
      assistantId,
      telemetryId,
      projectId,
      token,
      authId,
      e => {
        toast.error(e);
        setLoading(false);
      },
      () => {
        toast.success('Telemetry provider deleted successfully');
        get();
      },
    );
  };

  const filteredTelemetries = searchTerm.trim()
    ? action.telemetries.filter(row =>
        [
          row.getProvidertype(),
          String(row.getOptionsList().length),
          row.getEnabled() ? 'connected' : 'inactive',
        ]
          .join(' ')
          .toLowerCase()
          .includes(searchTerm.trim().toLowerCase()),
      )
    : action.telemetries;

  const selectedTelemetry = filteredTelemetries.find(
    row => row.getId() === selectedTelemetryId,
  );

  return (
    <div className="flex flex-col w-full flex-1 overflow-auto">
      <ConfirmDialogComponent />

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
          <h1 className="text-2xl font-light tracking-tight">Telemetry</h1>
        </div>
      </div>

      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={!!selectedTelemetry}
          totalSelected={selectedTelemetry ? 1 : 0}
          totalCount={filteredTelemetries.length}
          onCancel={() => setSelectedTelemetryId(null)}
          className="[&_[class*=divider]]:hidden [&_.cds--btn]:transition-colors [&_.cds--btn:hover]:!bg-primary [&_.cds--btn:hover]:!text-white"
        >
          {selectedTelemetry && (
            <>
              <TableBatchAction
                renderIcon={Edit}
                kind="ghost"
                onClick={() => {
                  navigation.goToEditAssistantTelemetry(
                    assistantId,
                    selectedTelemetry.getId(),
                  );
                  setSelectedTelemetryId(null);
                }}
              >
                Edit telemetry
              </TableBatchAction>
              <TableBatchAction
                renderIcon={TrashCan}
                kind="ghost"
                onClick={() => {
                  showDialog(() => deleteTelemetry(selectedTelemetry.getId()));
                  setSelectedTelemetryId(null);
                }}
              >
                Delete telemetry
              </TableBatchAction>
            </>
          )}
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search telemetry..."
            onChange={(e: any) => setSearchTerm(e.target?.value || '')}
          />
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={get}
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            onClick={() => navigation.goToCreateAssistantTelemetry(assistantId)}
          >
            Add telemetry
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>

      {loading ? (
        <div className="flex flex-col flex-1 items-center justify-center">
          <SectionLoader />
        </div>
      ) : action.telemetries.length > 0 && filteredTelemetries.length > 0 ? (
        <div className="overflow-auto flex-1">
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader className="!w-12" />
                <TableHeader>Provider</TableHeader>
                <TableHeader>Options</TableHeader>
                <TableHeader>Enabled</TableHeader>
                <TableHeader>Created</TableHeader>
                <TableHeader>Actions</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredTelemetries.map(row => {
                const providerType = row.getProvidertype();
                const providerName =
                  providerNameByCode.get(providerType) || providerType;
                const selected = row.getId() === selectedTelemetryId;
                return (
                  <TableRow
                    key={row.getId()}
                    isSelected={selected}
                    onClick={() =>
                      setSelectedTelemetryId(selected ? null : row.getId())
                    }
                    className="cursor-pointer"
                  >
                    <TableCell
                      className="!w-12 !pr-0"
                      onClick={e => e.stopPropagation()}
                    >
                      <RadioButton
                        id={`telemetry-select-${row.getId()}`}
                        name="telemetry-select"
                        labelText=""
                        hideLabel
                        checked={selected}
                        onChange={() =>
                          setSelectedTelemetryId(selected ? null : row.getId())
                        }
                      />
                    </TableCell>
                    <TableCell>{providerName}</TableCell>
                    <TableCell>{String(row.getOptionsList().length)}</TableCell>
                    <TableCell>
                      <Tag type={row.getEnabled() ? 'green' : 'gray'} size="sm">
                        {row.getEnabled() ? 'Yes' : 'No'}
                      </Tag>
                    </TableCell>
                    <TableCell>
                      {row.getCreateddate()
                        ? toHumanReadableDateTime(row.getCreateddate()!)
                        : '—'}
                    </TableCell>
                    <TableCell onClick={e => e.stopPropagation()}>
                      <div className="flex items-center gap-0">
                        <Button
                          hasIconOnly
                          renderIcon={Edit}
                          iconDescription="Edit telemetry"
                          kind="ghost"
                          size="sm"
                          onClick={() =>
                            navigation.goToEditAssistantTelemetry(
                              assistantId,
                              row.getId(),
                            )
                          }
                        />
                        <Button
                          hasIconOnly
                          renderIcon={TrashCan}
                          iconDescription="Delete telemetry"
                          kind="danger--ghost"
                          size="sm"
                          onClick={() =>
                            showDialog(() => deleteTelemetry(row.getId()))
                          }
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      ) : action.telemetries.length > 0 ? (
        <EmptyState
          icon={Activity}
          title="No telemetry providers found"
          subtitle="No telemetry provider matched your search."
        />
      ) : (
        <EmptyState
          icon={Activity}
          title="No telemetry providers"
          subtitle="Any telemetry providers you add will be listed here."
        />
      )}
    </div>
  );
};
