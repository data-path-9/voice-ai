import { FC } from 'react';
import { Endpoint } from '@rapidaai/react';
import { useEndpointPageStore } from '@/hooks';
import { nanoToMilli, toHumanReadableRelativeTime } from '@/utils/date';
import { useNavigate } from 'react-router-dom';
import { TableRow, TableCell, Tag, Link } from '@carbon/react';
import { ProviderTag } from '@/app/components/carbon/provider-tag';
import { IconOnlyButton } from '@/app/components/carbon/button';
import { Launch } from '@carbon/icons-react';
import { CarbonStatusIndicator } from '@/app/components/carbon/status-indicator';
import { VersionIndicator } from '@/app/components/indicators/version';

interface SingleEndpointProps {
  endpoint: Endpoint;
}

export const SingleEndpoint: FC<SingleEndpointProps> = ({ endpoint }) => {
  const endpointAction = useEndpointPageStore();
  const navigate = useNavigate();

  const getErrorRate = (endpoint: Endpoint) => {
    const errorCount = parseInt(
      endpoint.getEndpointanalytics()?.getErrorcount() ?? '0',
      10,
    );
    const totalCount = parseInt(
      endpoint.getEndpointanalytics()?.getCount() ?? '0',
      10,
    );
    if (errorCount === 0 || totalCount === 0) return 0;
    return Number((errorCount / totalCount) * 100).toFixed(2);
  };

  return (
    <TableRow>
      {endpointAction.visibleColumn('getStatus') && (
        <TableCell className="text-sm">
          <CarbonStatusIndicator state="DEPLOYED" />
        </TableCell>
      )}
      {endpointAction.visibleColumn('getName') && (
        <TableCell className="text-sm">
          <Link
            href={`/deployment/endpoint/${endpoint.getId()}`}
            className="!text-sm !inline-flex !items-center !gap-1"
          >
            <span>{endpoint?.getName()}</span>
            <Launch size={12} />
          </Link>
        </TableCell>
      )}
      {endpointAction.visibleColumn('action') && (
        <TableCell className="text-sm">
          <IconOnlyButton
            kind="ghost"
            size="md"
            renderIcon={Launch}
            iconDescription="View detail"
            onClick={() => navigate(`/deployment/endpoint/${endpoint.getId()}`)}
          />
        </TableCell>
      )}
      {endpointAction.visibleColumn('getVersion') && (
        <TableCell className="text-sm">
          <VersionIndicator id={endpoint.getEndpointprovidermodel()?.getId()!} />
        </TableCell>
      )}
      {endpointAction.visibleColumn('getTags') && (
        <TableCell className="text-sm">
          {endpoint.getEndpointtag()?.getTagList()?.length ? (
            <div className="flex flex-wrap gap-1">
              {endpoint.getEndpointtag()?.getTagList().map((tag, i) => (
                <Tag key={i} type="cool-gray" size="sm">{tag}</Tag>
              ))}
            </div>
          ) : (
            <span className="text-sm text-gray-400">—</span>
          )}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCount') && (
        <TableCell className="text-sm">
          <span className="tabular-nums text-blue-500 dark:text-blue-400">
            {endpoint.getEndpointanalytics()?.getCount()}
          </span>
        </TableCell>
      )}
      {endpointAction.visibleColumn('getErrorRate') && (
        <TableCell className="text-sm">
          <span className="tabular-nums text-red-500 dark:text-red-400">
            {getErrorRate(endpoint)}%
          </span>
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCurrentModel') && (
        <TableCell className="text-sm">
          <ProviderTag provider={endpoint.getEndpointprovidermodel()?.getModelprovidername()} />
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCost') && (
        <TableCell className="font-mono text-[13px] tabular-nums">
          ${((endpoint.getEndpointanalytics()?.getTotalinputcost() ?? 0) +
            (endpoint.getEndpointanalytics()?.getTotaloutputcost() ?? 0)).toFixed(4)}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getTotalToken') && (
        <TableCell className="text-sm tabular-nums">
          {endpoint.getEndpointanalytics()?.getTotaltoken()}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getP50') && (
        <TableCell className="font-mono text-[13px] tabular-nums">
          {nanoToMilli(endpoint.getEndpointanalytics()?.getP50latency())}ms
        </TableCell>
      )}
      {endpointAction.visibleColumn('getP99') && (
        <TableCell className="font-mono text-[13px] tabular-nums">
          {nanoToMilli(endpoint.getEndpointanalytics()?.getP99latency())}ms
        </TableCell>
      )}
      {endpointAction.visibleColumn('getMRR') && (
        <TableCell className="text-[13px]">
          {endpoint.getEndpointanalytics()?.getLastactivity() &&
          endpoint.getEndpointanalytics()?.getLastactivity()?.toDate().getTime()! > new Date('1970-01-01').getTime()
            ? toHumanReadableRelativeTime(endpoint.getEndpointanalytics()?.getLastactivity()!)
            : 'Not yet run'}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCreatedBy') && (
        <TableCell className="text-sm">
          <span className="capitalize text-sm">
            {endpoint.getEndpointprovidermodel()?.getCreateduser()?.getName()}
          </span>
        </TableCell>
      )}
    </TableRow>
  );
};
