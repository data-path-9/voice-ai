import {
  CUSTOM_TTS_DONE_REQUEST_EXAMPLE,
  CUSTOM_TTS_DSL_VARIABLES,
  CUSTOM_TTS_QUERY_PARAMS_EXAMPLE,
  CUSTOM_TTS_RESPONSE_PARSER_BINARY_EXAMPLE,
  CUSTOM_TTS_RESPONSE_PARSER_JSON_AUDIO_EXAMPLE,
  CUSTOM_TTS_TEXT_REQUEST_EXAMPLE,
} from '@/providers/custom-tts/template';

export type VoiceWebsocketEditorMode =
  | 'query_params'
  | 'text_request'
  | 'done_request'
  | 'response_parser';

export type VoiceWebsocketEditorSuggestion = {
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

const VARIABLE_DEFINITIONS: VariableDefinition[] = [
  {
    key: 'text',
    description: 'The text that should be synthesized in the websocket request.',
  },
  {
    key: 'message_id',
    description: 'The assistant message identifier used to correlate frames.',
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
];

const CAST_VALUES = ['string', 'number', 'boolean'] as const;

export const extractVoiceWebsocketVariableQuery = (
  linePrefix: string,
): string | null => {
  const match = linePrefix.match(VARIABLE_TRIGGER_REGEX);
  if (!match) return null;
  return match[1] || '';
};

export const extractVoiceWebsocketCastQuery = (
  linePrefix: string,
): string | null => {
  const match = linePrefix.match(CAST_TRIGGER_REGEX);
  if (!match) return null;
  return match[1] || '';
};

export const shouldAutoTriggerVoiceWebsocketSuggestions = (
  mode: VoiceWebsocketEditorMode,
  linePrefix: string,
): boolean => {
  const trimmed = linePrefix.trim();
  if (mode === 'response_parser') {
    return trimmed === '[';
  }

  return (
    trimmed === '{' ||
    extractVoiceWebsocketVariableQuery(linePrefix) !== null ||
    extractVoiceWebsocketCastQuery(linePrefix) !== null
  );
};

function getRequestSnippetSuggestion(
  mode: Extract<
    VoiceWebsocketEditorMode,
    'query_params' | 'text_request' | 'done_request'
  >,
): VoiceWebsocketEditorSuggestion {
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

function getResponseParserSnippetSuggestions(): VoiceWebsocketEditorSuggestion[] {
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

export const getVoiceWebsocketEditorSuggestions = (
  mode: VoiceWebsocketEditorMode,
  linePrefix: string,
): VoiceWebsocketEditorSuggestion[] => {
  if (mode === 'response_parser') {
    const trimmed = linePrefix.trim();
    if (trimmed === '' || trimmed.endsWith('[')) {
      return getResponseParserSnippetSuggestions();
    }
    return [];
  }

  const variableQuery = extractVoiceWebsocketVariableQuery(linePrefix);
  if (variableQuery !== null) {
    const normalizedQuery = variableQuery.toLowerCase();
    return VARIABLE_DEFINITIONS.filter(item =>
      item.key.toLowerCase().startsWith(normalizedQuery),
    ).map(item => ({
      label: item.key,
      insertText: item.key,
      description: item.description,
      detail: 'Custom TTS variable',
      kind: 'variable',
      query: variableQuery,
    }));
  }

  const castQuery = extractVoiceWebsocketCastQuery(linePrefix);
  if (castQuery !== null) {
    const normalizedQuery = castQuery.toLowerCase();
    return CAST_VALUES.filter(item => item.startsWith(normalizedQuery)).map(
      item => ({
        label: item,
        insertText: item,
        description: `Cast the resolved value to ${item}.`,
        detail: 'Custom TTS cast value',
        kind: 'value',
        query: castQuery,
      }),
    );
  }

  const trimmed = linePrefix.trim();
  if (trimmed === '' || trimmed.endsWith('{')) {
    return [getRequestSnippetSuggestion(mode)];
  }

  return [];
};

export const VOICE_WEBSOCKET_TEMPLATE_VARIABLE_KEYS = CUSTOM_TTS_DSL_VARIABLES;
