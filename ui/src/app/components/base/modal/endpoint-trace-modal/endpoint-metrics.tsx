import { Metric } from '@rapidaai/react';
import { Tooltip } from '@/app/components/tooltip';
import { InfoIcon } from 'lucide-react';
import { FC } from 'react';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { ChartLine } from '@carbon/icons-react';

export const EndpointMetrics: FC<{ metrics: Array<Metric> }> = ({
  metrics,
}) => {
  if (metrics.length <= 0)
    return (
      <EmptyState
        className="h-full min-h-[420px]"
        icon={ChartLine}
        title="No metrics found"
        subtitle="No metrics were recorded for this trace."
      />
    );
  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
      {metrics.map((x, idx) => (
        <div
          key={`metrics-idx-${idx}`}
          className="flex items-center justify-between h-12 px-4 gap-4"
        >
          <div className="flex items-center gap-1.5">
            <span className="text-xs font-medium uppercase tracking-[0.08em] text-gray-500 dark:text-gray-400 shrink-0">
              {x.getName()}
            </span>
            <Tooltip
              icon={<InfoIcon className="w-3.5 h-3.5 text-gray-400 shrink-0" />}
            >
              <p className="font-normal text-sm p-1 px-2">
                {x.getDescription()}
              </p>
            </Tooltip>
          </div>
          <div className="flex items-center shrink-0 ml-4">
            <MetricValue value={x.getValue()} />
          </div>
        </div>
      ))}
    </div>
  );
};

const MetricValue = ({ value }) => {
  if (
    typeof value === 'string' &&
    !isNaN(parseFloat(value)) &&
    parseFloat(value) < 1
  ) {
    const progress = Math.min(Math.max(parseFloat(value), 0), 100);
    return (
      <div className="w-24 bg-gray-200 dark:bg-gray-700 h-1.5">
        <div className="bg-primary h-1.5" style={{ width: `${progress}%` }} />
      </div>
    );
  } else if (value === 'false') {
    return (
      <svg
        viewBox="0 0 24 24"
        fill="currentColor"
        stroke="none"
        className="w-5 h-5 text-green-500"
      >
        <path d="M17 3.34a10 10 0 1 1 -14.995 8.984l-.005 -.324l.005 -.324a10 10 0 0 1 14.995 -8.336zm-1.293 5.953a1 1 0 0 0 -1.32 -.083l-.094 .083l-3.293 3.292l-1.293 -1.292l-.094 -.083a1 1 0 0 0 -1.403 1.403l.083 .094l2 2l.094 .083a1 1 0 0 0 1.226 0l.094 -.083l4 -4l.083 -.094a1 1 0 0 0 -.083 -1.32z" />
      </svg>
    );
  } else if (value === 'true') {
    return (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="currentColor"
        stroke="none"
        className="w-5 h-5 text-red-500"
      >
        <path d="M17 3.34a10 10 0 1 1 -14.995 8.984l-.005 -.324l.005 -.324a10 10 0 0 1 14.995 -8.336zm-6.489 5.8a1 1 0 0 0 -1.218 1.567l1.292 1.293l-1.292 1.293l-.083 .094a1 1 0 0 0 1.497 1.32l1.293 -1.292l1.293 1.292l.094 .083a1 1 0 0 0 1.32 -1.497l-1.292 -1.293l1.292 -1.293l.083 -.094a1 1 0 0 0 -1.497 -1.32l-1.293 1.292l-1.293 -1.292l-.094 -.083z" />
      </svg>
    );
  } else {
    return (
      <span className="text-gray-900 dark:text-gray-100 tabular-nums">
        {value}
      </span>
    );
  }
};
