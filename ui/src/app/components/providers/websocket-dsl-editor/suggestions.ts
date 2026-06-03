import {
  CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE,
  CUSTOM_TTS_DSL_VARIABLES,
  CUSTOM_TTS_QUERY_PARAMS_EXAMPLE,
  CUSTOM_TTS_REQUEST_RULES_DONE_EXAMPLE,
  CUSTOM_TTS_REQUEST_RULES_INTERRUPT_EXAMPLE,
  CUSTOM_TTS_RESPONSE_RULES_BINARY_EXAMPLE,
  CUSTOM_TTS_RESPONSE_RULES_JSON_AUDIO_EXAMPLE,
} from '@/providers/custom-tts/contract';
import {
  CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE,
  CUSTOM_STT_DSL_VARIABLES,
  CUSTOM_STT_QUERY_PARAMS_EXAMPLE,
  CUSTOM_STT_REQUEST_RULES_INTERRUPT_EXAMPLE,
  CUSTOM_STT_REQUEST_RULES_JSON_AUDIO_EXAMPLE,
  CUSTOM_STT_REQUEST_RULES_TURN_CHANGE_EXAMPLE,
  CUSTOM_STT_RESPONSE_RULES_EXAMPLE,
  CUSTOM_STT_RESPONSE_RULES_JSON_EXAMPLE,
  CUSTOM_STT_RESPONSE_RULES_NESTED_EXAMPLE,
} from '@/providers/custom-stt/contract';
import { WEBSOCKET_DSL_CAST_VALUES } from '@/providers/websocket-dsl/core';

export type WebsocketDslEditorProvider = 'custom-tts' | 'custom-stt';
export type WebsocketDslEditorMode =
  | 'query_params'
  | 'request_rules'
  | 'response_rules';

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

type PathDefinition = {
  key: string;
  description: string;
};

const VARIABLE_TRIGGER_REGEX = /"\$var"\s*:\s*"([a-zA-Z0-9_]*)$/;
const PATH_TRIGGER_REGEX = /"\$path"\s*:\s*"([a-zA-Z0-9_.]*)$/;
const CAST_TRIGGER_REGEX = /"\$cast"\s*:\s*"([a-zA-Z0-9_]*)$/;

const VARIABLE_DEFINITIONS: Record<
  WebsocketDslEditorProvider,
  VariableDefinition[]
> = {
  'custom-tts': [
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
  ],
  'custom-stt': [
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

const REQUEST_PATH_DEFINITIONS: Record<
  WebsocketDslEditorProvider,
  PathDefinition[]
> = {
  'custom-tts': [
    {
      key: 'config.voice.id',
      description: 'The configured custom TTS voice identifier.',
    },
    {
      key: 'config.model',
      description: 'The configured custom TTS model identifier.',
    },
    {
      key: 'config.language',
      description: 'The configured custom TTS language code.',
    },
    {
      key: 'config.audio.encoding',
      description: 'The configured output audio encoding.',
    },
    {
      key: 'config.audio.sample_rate',
      description: 'The configured output audio sample rate.',
    },
    {
      key: 'packet.kind',
      description: 'The normalized TTS packet kind currently being handled.',
    },
    {
      key: 'packet.message_id',
      description: 'The active assistant message identifier.',
    },
    {
      key: 'packet.text',
      description: 'The text payload for the current TTS packet.',
    },
  ],
  'custom-stt': [
    {
      key: 'config.model',
      description: 'The configured custom STT model identifier.',
    },
    {
      key: 'config.language',
      description: 'The configured custom STT language code.',
    },
    {
      key: 'config.audio.encoding',
      description: 'The configured input audio encoding.',
    },
    {
      key: 'config.audio.sample_rate',
      description: 'The configured input audio sample rate.',
    },
    {
      key: 'packet.kind',
      description: 'The normalized STT packet kind currently being handled.',
    },
    {
      key: 'packet.context_id',
      description: 'The active speech-turn context identifier.',
    },
    {
      key: 'packet.audio.bytes',
      description: 'The current audio packet as raw bytes-like data.',
    },
    {
      key: 'packet.audio.base64',
      description: 'The current audio packet encoded as base64.',
    },
    {
      key: 'packet.audio.pcm_base64',
      description: 'The current PCM audio packet encoded as base64.',
    },
    {
      key: 'packet.audio.wav_base64',
      description: 'The current audio packet wrapped in a WAV container and encoded as base64.',
    },
  ],
};

const RESPONSE_MODES: ReadonlySet<WebsocketDslEditorMode> = new Set([
  'response_rules',
]);

export const extractWebsocketDslVariableQuery = (
  linePrefix: string,
): string | null => {
  const match = linePrefix.match(VARIABLE_TRIGGER_REGEX);
  if (!match) return null;
  return match[1] || '';
};

export const extractWebsocketDslPathQuery = (
  linePrefix: string,
): string | null => {
  const match = linePrefix.match(PATH_TRIGGER_REGEX);
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
  if (RESPONSE_MODES.has(mode)) {
    return trimmed === '[';
  }

  if (mode === 'request_rules') {
    return (
      trimmed === '[' ||
      extractWebsocketDslPathQuery(linePrefix) !== null ||
      extractWebsocketDslCastQuery(linePrefix) !== null
    );
  }

  return (
    trimmed === '{' ||
    extractWebsocketDslVariableQuery(linePrefix) !== null ||
    extractWebsocketDslCastQuery(linePrefix) !== null
  );
};

function getQuerySnippetSuggestion(
  provider: WebsocketDslEditorProvider,
): WebsocketDslEditorSuggestion {
  if (provider === 'custom-stt') {
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
    label: 'Query params mapping',
    insertText: CUSTOM_TTS_QUERY_PARAMS_EXAMPLE,
    description:
      'Starter JSON mapping for websocket query params using $var and $cast.',
    detail: 'Custom TTS query snippet',
    kind: 'snippet',
  };
}

function getTtsRequestRuleSnippetSuggestions(): WebsocketDslEditorSuggestion[] {
  return [
    {
      label: 'Text JSON rule',
      insertText: CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE,
      description:
        'Send JSON websocket synthesis requests for normalized text packets.',
      detail: 'Custom TTS request rule snippet',
      kind: 'snippet',
    },
    {
      label: 'Text + done rules',
      insertText: CUSTOM_TTS_REQUEST_RULES_DONE_EXAMPLE,
      description:
        'Use a follow-up done packet to mark the end of a two-step TTS stream.',
      detail: 'Custom TTS request rule snippet',
      kind: 'snippet',
    },
    {
      label: 'Text + interrupt rules',
      insertText: CUSTOM_TTS_REQUEST_RULES_INTERRUPT_EXAMPLE,
      description:
        'Send an interrupt packet before the runtime closes the TTS websocket.',
      detail: 'Custom TTS request rule snippet',
      kind: 'snippet',
    },
  ];
}

function getSttRequestRuleSnippetSuggestions(): WebsocketDslEditorSuggestion[] {
  return [
    {
      label: 'Binary audio rule',
      insertText: CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE,
      description:
        'Send raw websocket binary frames for normalized audio packets.',
      detail: 'Custom STT request rule snippet',
      kind: 'snippet',
    },
    {
      label: 'JSON audio rule',
      insertText: CUSTOM_STT_REQUEST_RULES_JSON_AUDIO_EXAMPLE,
      description:
        'Send JSON websocket audio packets using packet.audio.base64 and config audio settings.',
      detail: 'Custom STT request rule snippet',
      kind: 'snippet',
    },
    {
      label: 'Turn-change start rule',
      insertText: CUSTOM_STT_REQUEST_RULES_TURN_CHANGE_EXAMPLE,
      description:
        'Start with a JSON turn-change packet, then stream raw binary audio.',
      detail: 'Custom STT request rule snippet',
      kind: 'snippet',
    },
    {
      label: 'Interrupt flush rule',
      insertText: CUSTOM_STT_REQUEST_RULES_INTERRUPT_EXAMPLE,
      description:
        'Add an interrupt packet that flushes the upstream STT stream.',
      detail: 'Custom STT request rule snippet',
      kind: 'snippet',
    },
  ];
}

function getResponseRuleSnippetSuggestions(
  provider: WebsocketDslEditorProvider,
): WebsocketDslEditorSuggestion[] {
  if (provider === 'custom-stt') {
    return [
      {
        label: 'Plain text transcript parser',
        insertText: CUSTOM_STT_RESPONSE_RULES_EXAMPLE,
        description:
          'Use when the provider emits transcript deltas as plain websocket text frames.',
        detail: 'Custom STT response rule snippet',
        kind: 'snippet',
      },
      {
        label: 'JSON transcript parser',
        insertText: CUSTOM_STT_RESPONSE_RULES_JSON_EXAMPLE,
        description:
          'Use when the provider returns JSON transcript frames with separate partial and final events.',
        detail: 'Custom STT response rule snippet',
        kind: 'snippet',
      },
      {
        label: 'Nested transcript parser',
        insertText: CUSTOM_STT_RESPONSE_RULES_NESTED_EXAMPLE,
        description:
          'Use when transcript data is nested under a result object.',
        detail: 'Custom STT response rule snippet',
        kind: 'snippet',
      },
    ];
  }

  return [
    {
      label: 'Binary audio parser',
      insertText: CUSTOM_TTS_RESPONSE_RULES_BINARY_EXAMPLE,
      description:
        'Use when audio arrives as binary frames and done/error arrives as JSON.',
      detail: 'Custom TTS response rule snippet',
      kind: 'snippet',
    },
    {
      label: 'JSON base64 audio parser',
      insertText: CUSTOM_TTS_RESPONSE_RULES_JSON_AUDIO_EXAMPLE,
      description: 'Use when audio arrives in JSON frames as a base64 payload.',
      detail: 'Custom TTS response rule snippet',
      kind: 'snippet',
    },
  ];
}

export const getWebsocketDslEditorSuggestions = (
  provider: WebsocketDslEditorProvider,
  mode: WebsocketDslEditorMode,
  linePrefix: string,
): WebsocketDslEditorSuggestion[] => {
  if (RESPONSE_MODES.has(mode)) {
    const trimmed = linePrefix.trim();
    if (trimmed === '' || trimmed.endsWith('[')) {
      return getResponseRuleSnippetSuggestions(provider);
    }
    return [];
  }

  if (mode === 'request_rules') {
    const pathQuery = extractWebsocketDslPathQuery(linePrefix);
    if (pathQuery !== null) {
      const normalizedQuery = pathQuery.toLowerCase();
      return REQUEST_PATH_DEFINITIONS[provider]
        .filter(item => item.key.toLowerCase().startsWith(normalizedQuery))
        .map(item => ({
          label: item.key,
          insertText: item.key,
          description: item.description,
          detail: 'Websocket DSL path',
          kind: 'variable',
          query: pathQuery,
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
    if (trimmed === '' || trimmed.endsWith('[')) {
      return provider === 'custom-stt'
        ? getSttRequestRuleSnippetSuggestions()
        : getTtsRequestRuleSnippetSuggestions();
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
    return [getQuerySnippetSuggestion(provider)];
  }

  return [];
};

export const WEBSOCKET_DSL_VARIABLE_KEYS = [
  ...CUSTOM_TTS_DSL_VARIABLES,
  ...CUSTOM_STT_DSL_VARIABLES,
];
