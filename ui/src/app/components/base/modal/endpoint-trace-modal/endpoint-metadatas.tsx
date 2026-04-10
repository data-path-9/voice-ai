import { Metadata } from '@rapidaai/react';
import { FC } from 'react';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { DataBase } from '@carbon/icons-react';

export const EndpointMetadatas: FC<{ metadata: Array<Metadata> }> = ({
  metadata,
}) => {
  if (metadata.length <= 0)
    return (
      <EmptyState
        className="h-full min-h-[420px]"
        icon={DataBase}
        title="No metadata found"
        subtitle="No metadata was recorded for this trace."
      />
    );
  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
      {metadata.map((x, idx) => (
        <div
          key={`metadata-idx-${idx}`}
          className="flex items-center justify-between h-10 px-4 text-sm gap-4"
        >
          <span className="text-xs font-medium uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400 shrink-0">
            {x.getKey()}
          </span>
          <span className="text-sm text-gray-900 dark:text-gray-100 font-mono truncate text-right">
            {x.getValue()}
          </span>
        </div>
      ))}
    </div>
  );
};
