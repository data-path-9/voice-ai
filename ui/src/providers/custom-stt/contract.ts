import {
  WebsocketDslCastTarget,
  WebsocketDslJsonValue,
  WebsocketDslPrimitive,
  WebsocketDslResponseParserRule,
  parseWebsocketDslResponseFrame,
  parseWebsocketDslResponseParser,
  renderWebsocketDslQueryParams,
  renderWebsocketDslRequestDefinition,
  validateWebsocketDslQueryParams,
  validateWebsocketDslRequestDefinition,
  validateWebsocketDslResponseParser,
} from '../websocket-dsl/core';

export const CUSTOM_STT_QUERY_PARAMS_KEY = 'listen.ws.query_params';
export const CUSTOM_STT_AUDIO_REQUEST_KEY = 'listen.ws.audio_request';
export const CUSTOM_STT_RESPONSE_PARSER_KEY = 'listen.ws.response_parser';

export const CUSTOM_STT_DSL_VARIABLES = [
  'audio',
  'model',
  'language',
  'encoding',
  'sample_rate',
] as const;

type CustomSttResponseEmitKey =
  | 'script'
  | 'confidence'
  | 'language'
  | 'interim'
  | 'error';

export type CustomSttDslVariable = (typeof CUSTOM_STT_DSL_VARIABLES)[number];
export type CustomSttDslCastTarget = WebsocketDslCastTarget;
export type CustomSttResponseFrameType = 'json' | 'text';
export type CustomSttPrimitive = WebsocketDslPrimitive;
export type CustomSttJsonValue = WebsocketDslJsonValue;

export interface CustomSttRequestContext {
  audio: string;
  model: string;
  language: string;
  encoding: string;
  sample_rate: string;
}

export type CustomSttResponseParserRule = WebsocketDslResponseParserRule<
  CustomSttResponseFrameType,
  CustomSttResponseEmitKey
>;

export type CustomSttParsedResponseFrame =
  | {
      kind: 'transcript';
      payload: unknown;
      script: string;
      confidence?: number;
      language?: string;
      interim: boolean;
    }
  | {
      kind: 'error';
      payload: unknown;
      error: unknown;
    }
  | {
      kind: 'message';
      payload: unknown;
      emitted?: Record<string, unknown>;
    };

const RESPONSE_VALIDATION_OPTIONS = {
  providerLabel: 'Custom STT',
  jsonErrorLabel: 'custom STT',
  supportedFrameTypes: ['json', 'text'] as const,
  supportedEmitKeys: [
    'script',
    'confidence',
    'language',
    'interim',
    'error',
  ] as const,
  allowedFrameExpressions: ['text'] as const,
};

const RESPONSE_PARSE_OPTIONS = {
  providerLabel: 'Custom STT',
  supportedFrameTypes: ['json', 'text'] as const,
  allowedFrameExpressions: ['text'] as const,
};

const VALIDATION_CONTEXT: CustomSttRequestContext = {
  audio: 'AAEC',
  model: 'nova-3',
  language: 'en-US',
  encoding: 'LINEAR16',
  sample_rate: '16000',
};

export const CUSTOM_STT_QUERY_PARAMS_EXAMPLE = `{
  "language": { "$var": "language" },
  "model": { "$var": "model" },
  "encoding": { "$var": "encoding" },
  "sample_rate": {
    "$cast": "number",
    "value": { "$var": "sample_rate" }
  }
}`;

export const CUSTOM_STT_AUDIO_REQUEST_EXAMPLE = `{
  "audio": { "$var": "audio" },
  "encoding": { "$var": "encoding" },
  "sample_rate": {
    "$cast": "number",
    "value": { "$var": "sample_rate" }
  }
}`;

export const CUSTOM_STT_RESPONSE_PARSER_EXAMPLE = `[
  {
    "when": { "frame": "text" },
    "emit": {
      "script": { "$frame": "text" },
      "language": "hi",
      "interim": true
    }
  }
]`;

export const CUSTOM_STT_RESPONSE_PARSER_JSON_EXAMPLE = `[
  {
    "when": { "frame": "json", "path": "type", "equals": "partial" },
    "emit": {
      "script": { "$path": "text" },
      "confidence": {
        "$cast": "number",
        "value": { "$path": "confidence" }
      },
      "language": { "$path": "language" },
      "interim": true
    }
  },
  {
    "when": { "frame": "json", "path": "type", "equals": "final" },
    "emit": {
      "script": { "$path": "text" },
      "confidence": {
        "$cast": "number",
        "value": { "$path": "confidence" }
      },
      "language": { "$path": "language" },
      "interim": false
    }
  },
  {
    "when": { "frame": "json", "path": "type", "equals": "error" },
    "emit": {
      "error": { "$path": "error.message" }
    }
  }
]`;

export const CUSTOM_STT_RESPONSE_PARSER_NESTED_EXAMPLE = `[
  {
    "when": { "frame": "json", "path": "result.final", "equals": false },
    "emit": {
      "script": { "$path": "result.transcript" },
      "interim": true
    }
  },
  {
    "when": { "frame": "json", "path": "result.final", "equals": true },
    "emit": {
      "script": { "$path": "result.transcript" },
      "confidence": {
        "$cast": "number",
        "value": { "$path": "result.confidence" }
      },
      "language": { "$path": "result.language" },
      "interim": false
    }
  }
]`;

function classifyEmittedResponse(
  payload: unknown,
  emitted: Record<string, unknown>,
): CustomSttParsedResponseFrame {
  if (
    emitted.error !== undefined &&
    emitted.error !== null &&
    emitted.error !== ''
  ) {
    return {
      kind: 'error',
      payload,
      error: emitted.error,
    };
  }

  if (typeof emitted.script === 'string') {
    const transcript: CustomSttParsedResponseFrame = {
      kind: 'transcript',
      payload,
      script: emitted.script,
      interim: emitted.interim === true,
    };

    if (
      typeof emitted.confidence === 'number' &&
      Number.isFinite(emitted.confidence)
    ) {
      transcript.confidence = emitted.confidence;
    }

    if (typeof emitted.language === 'string' && emitted.language.trim()) {
      transcript.language = emitted.language;
    }

    return transcript;
  }

  return {
    kind: 'message',
    payload,
    emitted,
  };
}

export function renderCustomSttRequestDefinition(
  definition: string,
  context: CustomSttRequestContext,
): unknown {
  return renderWebsocketDslRequestDefinition(definition, context);
}

export function renderCustomSttQueryParams(
  definition: string,
  context: CustomSttRequestContext,
): Record<string, string | number | boolean> {
  return renderWebsocketDslQueryParams(definition, context, 'Custom STT');
}

export function validateCustomSttRequestDefinition(
  value: string,
  definitionLabel = 'request definition',
): string | undefined {
  return validateWebsocketDslRequestDefinition(value, {
    providerLabel: 'Custom STT',
    definitionLabel,
    supportedVariables: CUSTOM_STT_DSL_VARIABLES,
    validationContext: VALIDATION_CONTEXT,
  });
}

export function validateCustomSttQueryParams(
  value: string,
  definitionLabel = 'query parameters',
): string | undefined {
  return validateWebsocketDslQueryParams(value, {
    providerLabel: 'Custom STT',
    definitionLabel,
    supportedVariables: CUSTOM_STT_DSL_VARIABLES,
    validationContext: VALIDATION_CONTEXT,
  });
}

export function validateCustomSttResponseParser(
  value: string,
): string | undefined {
  return validateWebsocketDslResponseParser(value, RESPONSE_VALIDATION_OPTIONS);
}

export function parseCustomSttResponseParser(
  value: string,
): CustomSttResponseParserRule[] {
  return parseWebsocketDslResponseParser(
    value,
    RESPONSE_VALIDATION_OPTIONS,
  ) as CustomSttResponseParserRule[];
}

export function parseCustomSttResponseFrame(
  frame: string | ArrayBuffer | Uint8Array,
  parser: CustomSttResponseParserRule[],
): CustomSttParsedResponseFrame {
  const parsed = parseWebsocketDslResponseFrame(frame, parser, RESPONSE_PARSE_OPTIONS);
  if (!parsed.emitted) {
    return {
      kind: 'message',
      payload: parsed.payload,
    };
  }

  return classifyEmittedResponse(parsed.payload, parsed.emitted);
}
