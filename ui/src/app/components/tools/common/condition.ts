import { Metadata } from '@rapidaai/react';
import { getOptionValue } from './utils';

export const ASSISTANT_CONDITION_JSON_KEY = 'tool.condition';
export const ASSISTANT_CONDITION_OPERATOR_SYMBOL = '=';
export const ASSISTANT_CONDITION_OPERATOR_OPTIONS = [
  { label: 'equals', value: ASSISTANT_CONDITION_OPERATOR_SYMBOL },
] as const;
export const ASSISTANT_CONDITION_KEYS = [
  'source',
  'conversation_mode',
  'direction',
] as const;
export type AssistantConditionKey = (typeof ASSISTANT_CONDITION_KEYS)[number];

export const ASSISTANT_CONDITION_SOURCES = [
  'all',
  'sdk',
  'web_plugin',
  'debugger',
  'phone',
] as const;
export type AssistantConditionSource =
  (typeof ASSISTANT_CONDITION_SOURCES)[number];
export const ASSISTANT_CONDITION_SOURCE_OPTIONS: Array<{
  label: string;
  value: AssistantConditionSource;
}> = [
    { label: 'All', value: 'all' },
    { label: 'SDK', value: 'sdk' },
    { label: 'Web Plugin', value: 'web_plugin' },
    { label: 'Debugger', value: 'debugger' },
    { label: 'Phone', value: 'phone' },
  ];

export const ASSISTANT_CONDITION_CONVERSATION_MODES = [
  'all',
  'text',
  'voice',
] as const;
export type AssistantConditionConversationMode =
  (typeof ASSISTANT_CONDITION_CONVERSATION_MODES)[number];
export const ASSISTANT_CONDITION_CONVERSATION_MODE_OPTIONS: Array<{
  label: string;
  value: AssistantConditionConversationMode;
}> = [
    { label: 'All', value: 'all' },
    { label: 'Text', value: 'text' },
    { label: 'Voice', value: 'voice' },
  ];

export const ASSISTANT_CONDITION_DIRECTIONS = [
  'both',
  'inbound',
  'outbound',
] as const;
export type AssistantConditionDirection =
  (typeof ASSISTANT_CONDITION_DIRECTIONS)[number];
export const ASSISTANT_CONDITION_DIRECTION_OPTIONS: Array<{
  label: string;
  value: AssistantConditionDirection;
}> = [
    { label: 'Both', value: 'both' },
    { label: 'Inbound', value: 'inbound' },
    { label: 'Outbound', value: 'outbound' },
  ];
export const ASSISTANT_CONDITION_KEY_OPTIONS: Array<{
  label: string;
  value: AssistantConditionKey;
}> = [
    { label: 'Source', value: 'source' },
    { label: 'Conversation mode', value: 'conversation_mode' },
    { label: 'Direction', value: 'direction' },
  ];
export const ASSISTANT_CONDITION_VALUE_OPTIONS_BY_KEY: Record<
  AssistantConditionKey,
  Array<{ label: string; value: string }>
> = {
  source: ASSISTANT_CONDITION_SOURCE_OPTIONS,
  conversation_mode: ASSISTANT_CONDITION_CONVERSATION_MODE_OPTIONS,
  direction: ASSISTANT_CONDITION_DIRECTION_OPTIONS,
};

export type ToolConditionKey = AssistantConditionKey;
export type ToolConditionSource = AssistantConditionSource;
export type ToolConditionConversationMode = AssistantConditionConversationMode;
export type ToolConditionDirection = AssistantConditionDirection;
export interface AssistantConditionEntry {
  key: AssistantConditionKey;
  condition: '=';
  value: string;
}
export type ToolConditionEntry = AssistantConditionEntry;

export const normalizeAssistantConditionDirection = (
  value?: string,
): AssistantConditionDirection => {
  if (
    value &&
    ASSISTANT_CONDITION_DIRECTIONS.includes(value as AssistantConditionDirection)
  ) {
    return value as AssistantConditionDirection;
  }
  return 'both';
};

export const normalizeAssistantConditionEntry = (
  raw?: Partial<ToolConditionEntry> | null,
): AssistantConditionEntry => {
  const key: AssistantConditionKey =
    raw?.key === 'conversation_mode'
      ? 'conversation_mode'
      : raw?.key === 'direction'
        ? 'direction'
        : 'source';
  if (key === 'conversation_mode') {
    return {
      key,
      condition: '=',
      value: normalizeToolConditionConversationMode(raw?.value),
    };
  }
  if (key === 'direction') {
    return {
      key,
      condition: '=',
      value: normalizeAssistantConditionDirection(raw?.value),
    };
  }
  return {
    key,
    condition: '=',
    value: normalizeToolConditionSource(raw?.value),
  };
};

export const normalizeAssistantConditionEntries = (
  raw: unknown,
): AssistantConditionEntry[] => {
  if (!Array.isArray(raw) || raw.length === 0) {
    return [{ key: 'source', condition: '=', value: 'all' }];
  }

  const normalized = raw
    .filter(
      item => typeof item === 'object' && item !== null && !Array.isArray(item),
    )
    .map(item =>
      normalizeAssistantConditionEntry(item as Partial<ToolConditionEntry>),
    );

  return normalized.length > 0
    ? normalized
    : [{ key: 'source', condition: '=', value: 'all' }];
};

const upsertMetadata = (
  parameters: Metadata[],
  key: string,
  value: string,
): Metadata[] => {
  const updated = [...parameters];
  const index = updated.findIndex(param => param.getKey() === key);
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  if (index >= 0) {
    updated[index] = metadata;
  } else {
    updated.push(metadata);
  }
  return updated;
};

export const normalizeToolConditionSource = (
  value?: string,
): ToolConditionSource => {
  if (
    value &&
    ASSISTANT_CONDITION_SOURCES.includes(value as ToolConditionSource)
  ) {
    return value as ToolConditionSource;
  }
  return 'all';
};

export const normalizeToolConditionConversationMode = (
  value?: string,
): ToolConditionConversationMode => {
  if (
    value &&
    ASSISTANT_CONDITION_CONVERSATION_MODES.includes(
      value as ToolConditionConversationMode,
    )
  ) {
    return value as ToolConditionConversationMode;
  }
  return 'all';
};

export const normalizeToolConditionDirection = (
  value?: string,
): ToolConditionDirection => {
  if (
    value &&
    ASSISTANT_CONDITION_DIRECTIONS.includes(value as ToolConditionDirection)
  ) {
    return value as ToolConditionDirection;
  }
  return 'both';
};

const normalizeConditionEntry = (
  raw?: Partial<ToolConditionEntry> | null,
): ToolConditionEntry => ({
  key:
    raw?.key === 'conversation_mode'
      ? 'conversation_mode'
      : raw?.key === 'direction'
        ? 'direction'
        : 'source',
  condition:
    raw?.condition === ASSISTANT_CONDITION_OPERATOR_SYMBOL
      ? ASSISTANT_CONDITION_OPERATOR_SYMBOL
      : ASSISTANT_CONDITION_OPERATOR_SYMBOL,
  value:
    raw?.key === 'conversation_mode'
      ? normalizeToolConditionConversationMode(raw?.value)
      : raw?.key === 'direction'
        ? normalizeToolConditionDirection(raw?.value)
        : normalizeToolConditionSource(raw?.value),
});

const defaultConditionEntries = (): ToolConditionEntry[] => [
  {
    key: 'source',
    condition: ASSISTANT_CONDITION_OPERATOR_SYMBOL,
    value: 'all',
  },
];

export const toToolConditionJson = (entries: ToolConditionEntry[]): string =>
  JSON.stringify(entries, null, 2);

const parseToolConditionJson = (
  jsonValue?: string,
): ToolConditionEntry[] | null => {
  if (!jsonValue) return null;
  try {
    const parsed = JSON.parse(jsonValue);
    if (Array.isArray(parsed)) {
      return parsed
        .filter(
          item =>
            typeof item === 'object' && item !== null && !Array.isArray(item),
        )
        .map(item =>
          normalizeConditionEntry(item as Partial<ToolConditionEntry>),
        );
    }

    return null;
  } catch {
    return null;
  }
};

const getPersistedToolConditionEntries = (
  parameters: Metadata[] | null | undefined,
): ToolConditionEntry[] | null => {
  const params = parameters || [];
  return parseToolConditionJson(
    getOptionValue(params, ASSISTANT_CONDITION_JSON_KEY),
  );
};

export const getToolConditionEntries = (
  parameters: Metadata[] | null | undefined,
): ToolConditionEntry[] => {
  const jsonEntries = getPersistedToolConditionEntries(parameters);
  if (jsonEntries && jsonEntries.length > 0) {
    return jsonEntries;
  }

  return defaultConditionEntries();
};

export const getToolConditionSource = (
  parameters: Metadata[] | null | undefined,
): ToolConditionSource => {
  const sourceCondition = getToolConditionEntries(parameters).find(
    entry => entry.key === 'source',
  );
  return normalizeToolConditionSource(sourceCondition?.value);
};

export const getToolConditionSourceLabel = (
  source: ToolConditionSource,
): string =>
  ASSISTANT_CONDITION_SOURCE_OPTIONS.find(option => option.value === source)
    ?.label || 'All';

export const withToolConditionSource = (
  parameters: Metadata[],
  source: ToolConditionSource,
): Metadata[] => {
  return withToolConditionEntries(parameters, [
    {
      key: 'source',
      condition: ASSISTANT_CONDITION_OPERATOR_SYMBOL,
      value: source,
    },
  ]);
};

export const withToolConditionEntries = (
  parameters: Metadata[],
  entries: ToolConditionEntry[],
): Metadata[] => {
  const normalizedEntries =
    entries.length > 0
      ? entries.map(entry => normalizeConditionEntry(entry))
      : defaultConditionEntries();
  const conditionJson = toToolConditionJson(normalizedEntries);

  return upsertMetadata(parameters, ASSISTANT_CONDITION_JSON_KEY, conditionJson);
};

export const withNormalizedToolCondition = (
  parameters: Metadata[],
  fallback?: Metadata[],
): Metadata[] => {
  const primary = getPersistedToolConditionEntries(parameters);
  const fallbackEntries = getPersistedToolConditionEntries(fallback || []);
  return withToolConditionEntries(
    parameters,
    primary && primary.length > 0
      ? primary
      : fallbackEntries && fallbackEntries.length > 0
        ? fallbackEntries
        : defaultConditionEntries(),
  );
};

export const validateToolConditionMetadata = (
  parameters: Metadata[],
): string | undefined => {
  const raw = getOptionValue(parameters, ASSISTANT_CONDITION_JSON_KEY);
  if (!raw || raw.trim() === '') {
    return 'Condition must be a valid JSON array.';
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return 'Condition must be a valid JSON array.';
  }

  if (!Array.isArray(parsed)) {
    return 'Condition must be a valid JSON array.';
  }

  if (parsed.length === 0) {
    return 'Condition must include at least one entry.';
  }

  for (const condition of parsed) {
    if (
      typeof condition !== 'object' ||
      condition === null ||
      Array.isArray(condition)
    ) {
      return 'Each condition must be an object with key, condition, and value.';
    }

    const entry = condition as Record<string, unknown>;
    if (
      typeof entry.key !== 'string' ||
      typeof entry.condition !== 'string' ||
      typeof entry.value !== 'string'
    ) {
      return 'Each condition entry must have string key, condition, and value.';
    }

    if (!ASSISTANT_CONDITION_KEYS.includes(entry.key as ToolConditionKey)) {
      return `Condition key must be one of: ${ASSISTANT_CONDITION_KEYS.join(', ')}.`;
    }

    if (entry.condition !== ASSISTANT_CONDITION_OPERATOR_SYMBOL) {
      return 'Condition operator must be "=".';
    }

    if (entry.key === 'source') {
      if (
        !ASSISTANT_CONDITION_SOURCES.includes(entry.value as ToolConditionSource)
      ) {
        return `Condition source must be one of: ${ASSISTANT_CONDITION_SOURCES.join(', ')}.`;
      }
    }

    if (entry.key === 'conversation_mode') {
      if (
        !ASSISTANT_CONDITION_CONVERSATION_MODES.includes(
          entry.value as ToolConditionConversationMode,
        )
      ) {
        return `Condition conversation_mode must be one of: ${ASSISTANT_CONDITION_CONVERSATION_MODES.join(', ')}.`;
      }
    }

    if (entry.key === 'direction') {
      if (
        !ASSISTANT_CONDITION_DIRECTIONS.includes(
          entry.value as ToolConditionDirection,
        )
      ) {
        return `Condition direction must be one of: ${ASSISTANT_CONDITION_DIRECTIONS.join(', ')}.`;
      }
    }
  }

  return undefined;
};
