import {
  CUSTOM_TTS_DONE_REQUEST_EXAMPLE,
  CUSTOM_TTS_DSL_VARIABLES,
  CUSTOM_TTS_QUERY_PARAMS_EXAMPLE,
  CUSTOM_TTS_RESPONSE_PARSER_BINARY_EXAMPLE,
  CUSTOM_TTS_RESPONSE_PARSER_JSON_AUDIO_EXAMPLE,
  CUSTOM_TTS_TEXT_REQUEST_EXAMPLE,
} from '@/providers/custom-tts/contract';
import {
  CUSTOM_STT_AUDIO_REQUEST_EXAMPLE,
  CUSTOM_STT_DSL_VARIABLES,
  CUSTOM_STT_QUERY_PARAMS_EXAMPLE,
  CUSTOM_STT_RESPONSE_PARSER_EXAMPLE,
  CUSTOM_STT_RESPONSE_PARSER_JSON_EXAMPLE,
  CUSTOM_STT_RESPONSE_PARSER_NESTED_EXAMPLE,
} from '@/providers/custom-stt/contract';
import { WEBSOCKET_DSL_CAST_VALUES } from '@/providers/websocket-dsl/core';

export type WebsocketDslEditorProvider = 'custom-tts' | 'custom-stt';
export type WebsocketDslEditorMode =
  | 'query_params'
  | 'text_request'
  | 'audio_request'
  | 'done_request'
  | 'response_parser';

export type WebsocketDslEditorSuggestion = {
  label: string;
  insertText: string;
  description: string;
  detail: string;
  kind: 'snippet' | 'variable' | 'value';
  query?: string;
};

type VariableDefinition = {
  key: string;
  description: string;
};

const VARIABLE_TRIGGER_REGEX = /"\$var"\s*:\s*"([a-zA-Z0-9_]*)$/;
const CAST_TRIGGER_REGEX = /"\$cast"\s*:\s*"([a-zA-Z0-9_]*)$/;

const VARIABLE_DEFINITIONS: Record<
  WebsocketDslEditorProvider,
  VariableDefinition[]
> = {
  'custom-tts': [
    {
      key: 'text',
      description:
        'The text that should be synthesized in the websocket request.',
    },
    {
      key: 'message_id',
      description:
        'The assistant message identifier used to correlate frames.',
    },
    {
      key: 'voice_id',
      description: 'The configured custom TTS voice identifier.',
    },
    {
      key: 'model',
      description: 'The configured custom TTS model identifier.',
    },
    {
      key: 'language',
      description: 'The configured language code.',
    },
    {
      key: 'encoding',
      description: 'The configured audio encoding.',
    },
    {
      key: 'sample_rate',
      description: 'The configured audio sample rate.',
    },
  ],
  'custom-stt': [
    {
      key: 'audio',
      description:
        'The encoded audio chunk value available when audio_request is used.',
    },
    {
      key: 'model',
      description: 'The configured custom STT model identifier.',
    },
    {
      key: 'language',
      description: 'The configured language code.',
    },
    {
      key: 'encoding',
      description: 'The configured audio encoding.',
    },
    {
      key: 'sample_rate',
      description: 'The configured audio sample rate.',
    },
  ],
};

export const extractWebsocketDslVariableQuery = (
  linePrefix: string,
): string | null => {
  const match = linePrefix.match(VARIABLE_TRIGGER_REGEX);
  if (!match) return null;
  return match[1] || '';
};

export const extractWebsocketDslCastQuery = (
  linePrefix: string,
): string | null => {
  const match = linePrefix.match(CAST_TRIGGER_REGEX);
  if (!match) return null;
  return match[1] || '';
};

export const shouldAutoTriggerWebsocketDslSuggestions = (
  mode: WebsocketDslEditorMode,
  linePrefix: string,
): boolean => {
  const trimmed = linePrefix.trim();
  if (mode === 'response_parser') {
    return trimmed === '[';
  }

  return (
    trimmed === '{' ||
    extractWebsocketDslVariableQuery(linePrefix) !== null ||
    extractWebsocketDslCastQuery(linePrefix) !== null
  );
};

function getRequestSnippetSuggestion(
  provider: WebsocketDslEditorProvider,
  mode: Extract<
    WebsocketDslEditorMode,
    'query_params' | 'text_request' | 'audio_request' | 'done_request'
  >,
): WebsocketDslEditorSuggestion {
  if (provider === 'custom-stt') {
    if (mode === 'query_params') {
      return {
        label: 'Query params mapping',
        insertText: CUSTOM_STT_QUERY_PARAMS_EXAMPLE,
        description:
          'Starter JSON mapping for websocket query params using $var and $cast.',
        detail: 'Custom STT query snippet',
        kind: 'snippet',
      };
    }

    return {
      label: 'Audio request mapping',
      insertText: CUSTOM_STT_AUDIO_REQUEST_EXAMPLE,
      description:
        'Starter JSON mapping for websocket audio packets using the STT DSL.',
      detail: 'Custom STT request snippet',
      kind: 'snippet',
    };
  }

  if (mode === 'query_params') {
    return {
      label: 'Query params mapping',
      insertText: CUSTOM_TTS_QUERY_PARAMS_EXAMPLE,
      description:
        'Starter JSON mapping for websocket query params using $var and $cast.',
      detail: 'Custom TTS query snippet',
      kind: 'snippet',
    };
  }

  if (mode === 'text_request') {
    return {
      label: 'Text request mapping',
      insertText: CUSTOM_TTS_TEXT_REQUEST_EXAMPLE,
      description:
        'Starter JSON mapping for the primary speak request packet.',
      detail: 'Custom TTS request snippet',
      kind: 'snippet',
    };
  }

  return {
    label: 'Done request mapping',
    insertText: CUSTOM_TTS_DONE_REQUEST_EXAMPLE,
    description: 'Starter JSON mapping for the optional follow-up done packet.',
    detail: 'Custom TTS request snippet',
    kind: 'snippet',
  };
}

function getResponseParserSnippetSuggestions(
  provider: WebsocketDslEditorProvider,
): WebsocketDslEditorSuggestion[] {
  if (provider === 'custom-stt') {
    return [
      {
        label: 'Plain text transcript parser',
        insertText: CUSTOM_STT_RESPONSE_PARSER_EXAMPLE,
        description:
          'Use when the provider emits transcript deltas as plain websocket text frames.',
        detail: 'Custom STT response parser snippet',
        kind: 'snippet',
      },
      {
        label: 'JSON transcript parser',
        insertText: CUSTOM_STT_RESPONSE_PARSER_JSON_EXAMPLE,
        description:
          'Use when the provider returns JSON transcript frames with separate partial and final events.',
        detail: 'Custom STT response parser snippet',
        kind: 'snippet',
      },
      {
        label: 'Nested transcript parser',
        insertText: CUSTOM_STT_RESPONSE_PARSER_NESTED_EXAMPLE,
        description:
          'Use when transcript data is nested under a result object.',
        detail: 'Custom STT response parser snippet',
        kind: 'snippet',
      },
    ];
  }

  return [
    {
      label: 'Binary audio parser',
      insertText: CUSTOM_TTS_RESPONSE_PARSER_BINARY_EXAMPLE,
      description:
        'Use when audio arrives as binary frames and done/error arrives as JSON.',
      detail: 'Custom TTS response parser snippet',
      kind: 'snippet',
    },
    {
      label: 'JSON base64 audio parser',
      insertText: CUSTOM_TTS_RESPONSE_PARSER_JSON_AUDIO_EXAMPLE,
      description:
        'Use when audio arrives in JSON frames as a base64 payload.',
      detail: 'Custom TTS response parser snippet',
      kind: 'snippet',
    },
  ];
}

export const getWebsocketDslEditorSuggestions = (
  provider: WebsocketDslEditorProvider,
  mode: WebsocketDslEditorMode,
  linePrefix: string,
): WebsocketDslEditorSuggestion[] => {
  if (mode === 'response_parser') {
    const trimmed = linePrefix.trim();
    if (trimmed === '' || trimmed.endsWith('[')) {
      return getResponseParserSnippetSuggestions(provider);
    }
    return [];
  }

  const variableQuery = extractWebsocketDslVariableQuery(linePrefix);
  if (variableQuery !== null) {
    const normalizedQuery = variableQuery.toLowerCase();
    return VARIABLE_DEFINITIONS[provider]
      .filter(item => item.key.toLowerCase().startsWith(normalizedQuery))
      .map(item => ({
        label: item.key,
        insertText: item.key,
        description: item.description,
        detail:
          provider === 'custom-stt'
            ? 'Custom STT variable'
            : 'Custom TTS variable',
        kind: 'variable',
        query: variableQuery,
      }));
  }

  const castQuery = extractWebsocketDslCastQuery(linePrefix);
  if (castQuery !== null) {
    const normalizedQuery = castQuery.toLowerCase();
    return WEBSOCKET_DSL_CAST_VALUES.filter(item =>
      item.startsWith(normalizedQuery),
    ).map(item => ({
      label: item,
      insertText: item,
      description: `Cast the resolved value to ${item}.`,
      detail: 'Websocket DSL cast value',
      kind: 'value',
      query: castQuery,
    }));
  }

  const trimmed = linePrefix.trim();
  if (trimmed === '' || trimmed.endsWith('{')) {
    if (mode === 'response_parser') {
      return [];
    }

    return [
      getRequestSnippetSuggestion(
        provider,
        mode as Extract<
          WebsocketDslEditorMode,
          'query_params' | 'text_request' | 'audio_request' | 'done_request'
        >,
      ),
    ];
  }

  return [];
};

export const WEBSOCKET_DSL_VARIABLE_KEYS = [
  ...CUSTOM_TTS_DSL_VARIABLES,
  ...CUSTOM_STT_DSL_VARIABLES,
];
