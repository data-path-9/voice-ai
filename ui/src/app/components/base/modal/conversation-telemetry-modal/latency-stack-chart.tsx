import React from 'react';
import { Loading } from '@carbon/react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip as RechartsTooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { LATENCY_METRIC_META, LATENCY_STACK_ORDER } from './utils';
import type { LatencyMetricName, LatencySeriesPoint } from './utils';

type LatencyStackChartProps = {
  isLoading: boolean;
  latencySeries: LatencySeriesPoint[];
};

type LatencyTooltipPayloadItem = {
  dataKey?: string | number;
  value?: number | string;
  color?: string;
  payload?: LatencySeriesPoint;
};

const getVisibleLatencyTooltipItems = (
  payload: LatencyTooltipPayloadItem[] | undefined,
  metricOrder: LatencyMetricName[],
): LatencyTooltipPayloadItem[] => {
  if (!payload?.length) return [];
  const metricOrderIndex = new Map(
    metricOrder.map((metricName, index) => [metricName, index]),
  );

  return Array.from(
    payload.reduce(
      (acc, item) => acc.set(String(item.dataKey), item),
      new Map<string, LatencyTooltipPayloadItem>(),
    ),
  )
    .map(([, item]) => item)
    .filter(item => Number.isFinite(Number(item?.value)))
    .sort(
      (a, b) =>
        (metricOrderIndex.get(String(a.dataKey) as LatencyMetricName) ??
          Number.MAX_SAFE_INTEGER) -
        (metricOrderIndex.get(String(b.dataKey) as LatencyMetricName) ??
          Number.MAX_SAFE_INTEGER),
    );
};

const buildAverageLatencyByMetric = (
  latencySeries: LatencySeriesPoint[],
  metricNames: LatencyMetricName[],
): Record<LatencyMetricName, number> =>
  metricNames.reduce(
    (acc, metricName) => {
      const values = latencySeries
        .map(point => point[metricName])
        .filter(
          (value): value is number =>
            typeof value === 'number' && Number.isFinite(value),
        );
      acc[metricName] =
        values.length > 0
          ? Math.round(
              values.reduce((sum, current) => sum + current, 0) / values.length,
            )
          : 0;
      return acc;
    },
    {} as Record<LatencyMetricName, number>,
  );

export function LatencyStackChart(props: LatencyStackChartProps) {
  const { isLoading, latencySeries } = props;
  const latencyMetricNames = LATENCY_STACK_ORDER;
  const stackOrderText = latencyMetricNames
    .map(metricName => LATENCY_METRIC_META[metricName].shortLabel)
    .join(' -> ');
  const avgLatencyByMetric = buildAverageLatencyByMetric(
    latencySeries,
    latencyMetricNames,
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loading withOverlay={false} small />
      </div>
    );
  }

  if (latencySeries.length === 0) {
    return (
      <div className="flex items-center justify-center py-16 text-gray-400 dark:text-gray-500 text-sm">
        No latency metrics found
      </div>
    );
  }

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      <div className="flex flex-wrap items-center gap-6 px-4 pt-3 pb-2">
        {latencyMetricNames.map(metricName => (
          <div key={metricName}>
            <p className="text-[10px] text-gray-400 uppercase">
              {LATENCY_METRIC_META[metricName].shortLabel}
            </p>
            <p className="text-xl font-light tabular-nums">
              {avgLatencyByMetric[metricName]}{' '}
              <span className="text-xs text-gray-500">ms</span>
            </p>
          </div>
        ))}
      </div>
      <div className="flex-1 min-h-0 px-2">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart
            data={latencySeries}
            margin={{ top: 4, right: 4, left: 0, bottom: 0 }}
          >
            <CartesianGrid
              strokeDasharray="3 3"
              vertical={false}
              strokeOpacity={0.25}
            />
            <XAxis
              dataKey="sequence"
              tickLine={false}
              axisLine={false}
              tick={false}
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              width={42}
              tick={{ fontSize: 11, fill: '#9ca3af' }}
            />
            {latencyMetricNames.map(metricName => (
              <Area
                key={metricName}
                type="monotone"
                dataKey={metricName}
                stackId="latency"
                stroke={LATENCY_METRIC_META[metricName].color}
                strokeWidth={1.5}
                fill={LATENCY_METRIC_META[metricName].color}
                fillOpacity={LATENCY_METRIC_META[metricName].fillOpacity}
                connectNulls
                dot={false}
              />
            ))}
            <RechartsTooltip
              content={({ active, payload }) => {
                const tooltipPayload = (payload ||
                  []) as LatencyTooltipPayloadItem[];
                if (!active || tooltipPayload.length === 0) return null;
                const point = tooltipPayload[0]?.payload;
                const visiblePayload = getVisibleLatencyTooltipItems(
                  tooltipPayload,
                  latencyMetricNames,
                );
                if (visiblePayload.length === 0) return null;

                const total = visiblePayload.reduce(
                  (sum, item) => sum + Number(item.value || 0),
                  0,
                );

                return (
                  <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 shadow-lg px-3 py-2 text-sm min-w-[180px]">
                    <p className="text-gray-400 text-xs mb-1.5">
                      {point?.contextId
                        ? `${point.timeLabel} • ${point.contextId}`
                        : point?.timeLabel || ''}
                    </p>
                    <p className="text-[11px] text-gray-500 mb-1.5">
                      Stack: {stackOrderText}
                    </p>
                    {visiblePayload.map(item => {
                      const metricName = item.dataKey as LatencyMetricName;
                      const meta = LATENCY_METRIC_META[metricName];
                      const orderPosition =
                        latencyMetricNames.indexOf(metricName) + 1;
                      return (
                        <div
                          key={String(item.dataKey)}
                          className="flex items-center gap-2"
                        >
                          <div
                            className="w-2 h-2"
                            style={{
                              backgroundColor: item.color || meta?.color,
                            }}
                          />
                          <span className="text-gray-600 dark:text-gray-300 uppercase text-xs">
                            {orderPosition}.{' '}
                            {meta?.shortLabel || String(item.dataKey)}
                          </span>
                          <span className="ml-auto font-semibold tabular-nums">
                            {Number(item.value)} ms
                          </span>
                        </div>
                      );
                    })}
                    <div className="mt-2 border-t border-gray-200 dark:border-gray-700 pt-1.5 flex items-center">
                      <span className="text-[11px] uppercase tracking-wide text-gray-500">
                        Total
                      </span>
                      <span className="ml-auto font-semibold tabular-nums">
                        {total} ms
                      </span>
                    </div>
                  </div>
                );
              }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
      <div className="flex flex-wrap justify-center gap-4 px-4 pb-3 text-xs">
        {latencyMetricNames.map((metricName, index) => (
          <div
            key={`${metricName}-legend`}
            className="flex items-center gap-1.5"
          >
            <div
              className="w-3 h-0.5"
              style={{
                backgroundColor: LATENCY_METRIC_META[metricName].color,
              }}
            />
            {index + 1}. {LATENCY_METRIC_META[metricName].shortLabel}
          </div>
        ))}
      </div>
    </div>
  );
}
