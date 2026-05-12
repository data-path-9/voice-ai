import { FC, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Loading,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@carbon/react';
import { Copy, Checkmark } from '@carbon/icons-react';
import {
  ConnectionConfig,
  GetAllAssistantApiDeployment,
  GetAllAssistantDebuggerDeployment,
  GetAllAssistantDeploymentRequest,
  GetAllAssistantPhoneDeployment,
  GetAllAssistantWebpluginDeployment,
  Paginate,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { toHumanReadableDateTime } from '@/utils/date';
import { connectionConfig } from '@/configs';

export type AssistantDeploymentType = 'debugger' | 'api' | 'web' | 'phone';

interface AssistantDeploymentVersionsModalProps extends ModalProps {
  assistantId: string;
  deploymentType: AssistantDeploymentType | null;
  authId: string;
  token: string;
  projectId: string;
}

type DeploymentVersionRow = {
  id: string;
  createdDate: unknown;
};

const labelByType: Record<AssistantDeploymentType, string> = {
  api: 'SDK / API',
  debugger: 'Debugger',
  phone: 'Phone Call',
  web: 'Web Widget',
};

export const AssistantDeploymentVersionsModal: FC<
  AssistantDeploymentVersionsModalProps
> = ({
  modalOpen,
  setModalOpen,
  assistantId,
  deploymentType,
  authId,
  token,
  projectId,
}) => {
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [rows, setRows] = useState<DeploymentVersionRow[]>([]);
  const [copiedVersion, setCopiedVersion] = useState<string | null>(null);

  const title = deploymentType
    ? `${labelByType[deploymentType]} versions`
    : 'Deployment versions';

  const auth = useMemo(
    () =>
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId,
      }),
    [token, authId, projectId],
  );

  useEffect(() => {
    if (!modalOpen || !deploymentType) return;

    let isActive = true;
    setLoading(true);
    setErrorMessage('');
    setRows([]);

    const request = new GetAllAssistantDeploymentRequest();
    request.setAssistantid(assistantId);
    if ((request as any).setPaginate) {
      const paginate = new Paginate();
      paginate.setPage(1);
      paginate.setPagesize(100);
      (request as any).setPaginate(paginate);
    }

    const fetchByType = {
      api: GetAllAssistantApiDeployment,
      debugger: GetAllAssistantDebuggerDeployment,
      phone: GetAllAssistantPhoneDeployment,
      web: GetAllAssistantWebpluginDeployment,
    } as const;

    fetchByType[deploymentType](connectionConfig, request, auth)
      .then(response => {
        if (!isActive) return;
        if (!response?.getSuccess()) {
          setErrorMessage(
            response?.getError?.()?.getHumanmessage?.() ||
              'Unable to load deployment versions.',
          );
          return;
        }

        const data = (response as any).getDataList?.() || [];
        const mapped: DeploymentVersionRow[] = data.map((d: any) => ({
          id: d.getId?.() || '',
          createdDate: d.getCreateddate?.(),
        }));
        setRows(mapped);
      })
      .catch(() => {
        if (!isActive) return;
        setErrorMessage('Unable to load deployment versions.');
      })
      .finally(() => {
        if (!isActive) return;
        setLoading(false);
      });

    return () => {
      isActive = false;
    };
  }, [modalOpen, deploymentType, assistantId, auth]);

  const copyVersion = (id: string) => {
    const version = `vrsn_${id}`;
    navigator.clipboard.writeText(version);
    setCopiedVersion(version);
    toast.success('Version copied.');
    setTimeout(() => setCopiedVersion(null), 1500);
  };

  return (
    <RightSideModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      className="w-[580px]"
      label="Deployment"
      title={title}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loading withOverlay={false} />
          </div>
        ) : errorMessage ? (
          <p className="text-sm text-red-600 dark:text-red-400">{errorMessage}</p>
        ) : rows.length === 0 ? (
          <p className="text-sm text-gray-600 dark:text-gray-300">
            No versions found.
          </p>
        ) : (
          <div className="overflow-auto flex-1">
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Version</TableHeader>
                  <TableHeader>Date</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {rows.map(row => (
                  <TableRow key={row.id}>
                    <TableCell className="!font-mono !text-xs">
                      <span className="inline-flex items-center gap-1">
                        {`vrsn_${row.id}`}
                        <Button
                          hasIconOnly
                          renderIcon={
                            copiedVersion === `vrsn_${row.id}` ? Checkmark : Copy
                          }
                          iconDescription="Copy version id"
                          kind="ghost"
                          size="sm"
                          onClick={() => copyVersion(row.id)}
                          className="!min-h-0 !p-1"
                        />
                      </span>
                    </TableCell>
                    <TableCell className="!text-xs whitespace-nowrap">
                      {row.createdDate
                        ? toHumanReadableDateTime(row.createdDate as any)
                        : '—'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </RightSideModal>
  );
};
