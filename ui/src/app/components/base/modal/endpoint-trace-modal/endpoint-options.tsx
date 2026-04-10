import { Metadata } from '@rapidaai/react';
import { FC } from 'react';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { ModelAlt } from '@carbon/icons-react';

export const EndpointOptions: FC<{ options: Array<Metadata> }> = ({
  options,
}) => {
  if (options.length <= 0)
    return (
      <EmptyState
        className="h-full min-h-[420px]"
        icon={ModelAlt}
        title="No model options found"
        subtitle="No model execution options were recorded for this trace."
      />
    );
  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
      {options.map((x, idx) => (
        <div
          key={`options-idx-${idx}`}
          className="flex items-start justify-between px-4 py-3 gap-4"
        >
          <span className="text-xs font-medium uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400 shrink-0 pt-0.5">
            {x.getKey()}
          </span>
          <pre className="text-xs text-gray-900 dark:text-gray-100 font-mono text-right whitespace-pre-wrap break-all max-w-[70%] m-0 leading-5">
            {x.getValue()}
          </pre>
        </div>
      ))}
    </div>
  );
};
