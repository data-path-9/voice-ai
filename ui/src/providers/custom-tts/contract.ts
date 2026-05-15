import { MetadataLike } from '../config-loader';
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

export const CUSTOM_TTS_QUERY_PARAMS_KEY = 'speak.ws.query_params';
export const CUSTOM_TTS_TEXT_REQUEST_KEY = 'speak.ws.text_request';
export const CUSTOM_TTS_DONE_REQUEST_KEY = 'speak.ws.done_request';
export const CUSTOM_TTS_RESPONSE_PARSER_KEY = 'speak.ws.response_parser';

export const CUSTOM_TTS_DSL_VARIABLES = [
  'text',
  'message_id',
  'voice_id',
  'model',
  'language',
  'encoding',
  'sample_rate',
] as const;

type CustomTtsResponseEmitKey = 'audio' | 'message_id' | 'done' | 'error';

export type CustomTtsDslVariable = (typeof CUSTOM_TTS_DSL_VARIABLES)[number];
export type CustomTtsFlowMode = 'one-shot' | 'two-step';
export type CustomTtsDslCastTarget = WebsocketDslCastTarget;
export type CustomTtsResponseFrameType = 'binary' | 'json';
export type CustomTtsPrimitive = WebsocketDslPrimitive;
export type CustomTtsJsonValue = WebsocketDslJsonValue;

export interface CustomTtsRequestContext {
  text: string;
  message_id: string;
  voice_id: string;
  model: string;
  language: string;
  encoding: string;
  sample_rate: string;
}

export type CustomTtsResponseParserRule = WebsocketDslResponseParserRule<
  CustomTtsResponseFrameType,
  CustomTtsResponseEmitKey
>;

export type CustomTtsParsedResponseFrame =
  | {
      kind: 'audio';
      messageId?: string;
      payload?: unknown;
      audio: Uint8Array | string;
    }
  | {
      kind: 'done';
      messageId?: string;
      payload: unknown;
      done: true;
    }
  | {
      kind: 'error';
      messageId?: string;
      payload: unknown;
      error: unknown;
      done?: boolean;
    }
  | {
      kind: 'message';
      messageId?: string;
      payload: unknown;
      emitted?: Record<string, unknown>;
    };

const RESPONSE_VALIDATION_OPTIONS = {
  providerLabel: 'Custom TTS',
  jsonErrorLabel: 'custom TTS',
  supportedFrameTypes: ['binary', 'json'] as const,
  supportedEmitKeys: ['audio', 'message_id', 'done', 'error'] as const,
  allowedFrameExpressions: ['binary'] as const,
  allowDecodeBase64: true,
};

const RESPONSE_PARSE_OPTIONS = {
  providerLabel: 'Custom TTS',
  supportedFrameTypes: ['binary', 'json'] as const,
  allowedFrameExpressions: ['binary'] as const,
  allowDecodeBase64: true,
};

const VALIDATION_CONTEXT: CustomTtsRequestContext = {
  text: 'Hello world',
  message_id: 'msg_123',
  voice_id: 'voice_123',
  model: 'model_123',
  language: 'en-US',
  encoding: 'LINEAR16',
  sample_rate: '16000',
};

export const CUSTOM_TTS_QUERY_PARAMS_EXAMPLE = `{
  "language": { "$var": "language" },
  "model": { "$var": "model" },
  "voice": { "$var": "voice_id" },
  "message_id": { "$var": "message_id" },
  "sample_rate": {
    "$cast": "number",
    "value": { "$var": "sample_rate" }
  }
}`;

export const CUSTOM_TTS_TEXT_REQUEST_EXAMPLE = `{
  "text": { "$var": "text" },
  "voice_id": { "$var": "voice_id" },
  "message_id": { "$var": "message_id" },
  "model": { "$var": "model" },
  "language": { "$var": "language" },
  "audio": {
    "encoding": { "$var": "encoding" },
    "sample_rate": {
      "$cast": "number",
      "value": { "$var": "sample_rate" }
    }
  }
}`;

export const CUSTOM_TTS_DONE_REQUEST_EXAMPLE = `{
  "type": "done",
  "message_id": { "$var": "message_id" }
}`;

export const CUSTOM_TTS_RESPONSE_PARSER_BINARY_EXAMPLE = `[
  {
    "when": { "frame": "binary" },
    "emit": {
      "audio": { "$frame": "binary" }
    }
  },
  {
    "when": { "frame": "json", "path": "type", "equals": "done" },
    "emit": {
      "message_id": { "$path": "message_id" },
      "done": true
    }
  },
  {
    "when": { "frame": "json", "path": "type", "equals": "error" },
    "emit": {
      "message_id": { "$path": "message_id" },
      "error": { "$path": "error.message" },
      "done": true
    }
  }
]`;

export const CUSTOM_TTS_RESPONSE_PARSER_JSON_AUDIO_EXAMPLE = `[
  {
    "when": { "frame": "json", "path": "type", "equals": "chunk" },
    "emit": {
      "audio": {
        "$decode": "base64",
        "value": { "$path": "audio" }
      },
      "message_id": { "$path": "message_id" }
    }
  },
  {
    "when": { "frame": "json", "path": "type", "equals": "done" },
    "emit": {
      "message_id": { "$path": "message_id" },
      "done": true
    }
  },
  {
    "when": { "frame": "json", "path": "type", "equals": "error" },
    "emit": {
      "message_id": { "$path": "message_id" },
      "error": { "$path": "error.message" },
      "done": true
    }
  }
]`;

function classifyEmittedResponse(
  payload: unknown,
  emitted: Record<string, unknown>,
): CustomTtsParsedResponseFrame {
  const messageId =
    typeof emitted.message_id === 'string' && emitted.message_id.trim()
      ? emitted.message_id
      : undefined;

  if (
    emitted.error !== undefined &&
    emitted.error !== null &&
    emitted.error !== ''
  ) {
    return {
      kind: 'error',
      messageId,
      payload,
      error: emitted.error,
      done: emitted.done === true ? true : undefined,
    };
  }

  if (
    emitted.audio !== undefined &&
    (typeof emitted.audio === 'string' || emitted.audio instanceof Uint8Array)
  ) {
    return {
      kind: 'audio',
      messageId,
      payload,
      audio: emitted.audio,
    };
  }

  if (emitted.done === true) {
    return {
      kind: 'done',
      messageId,
      payload,
      done: true,
    };
  }

  return {
    kind: 'message',
    messageId,
    payload,
    emitted,
  };
}

export function resolveCustomTtsFlowMode(
  doneRequest?: string | null,
): CustomTtsFlowMode {
  return doneRequest?.trim() ? 'two-step' : 'one-shot';
}

export function getCustomTtsFlowMode(
  metadata: MetadataLike[],
): CustomTtsFlowMode {
  const doneRequest =
    metadata.find(item => item.getKey() === CUSTOM_TTS_DONE_REQUEST_KEY)?.getValue() ??
    '';
  return resolveCustomTtsFlowMode(doneRequest);
}

export function renderCustomTtsRequestDefinition(
  definition: string,
  context: CustomTtsRequestContext,
): unknown {
  return renderWebsocketDslRequestDefinition(definition, context);
}

export function renderCustomTtsQueryParams(
  definition: string,
  context: CustomTtsRequestContext,
): Record<string, string | number | boolean> {
  return renderWebsocketDslQueryParams(definition, context, 'Custom TTS');
}

export function validateCustomTtsRequestDefinition(
  value: string,
  definitionLabel = 'request definition',
): string | undefined {
  return validateWebsocketDslRequestDefinition(value, {
    providerLabel: 'Custom TTS',
    definitionLabel,
    supportedVariables: CUSTOM_TTS_DSL_VARIABLES,
    validationContext: VALIDATION_CONTEXT,
  });
}

export function validateCustomTtsQueryParams(
  value: string,
  definitionLabel = 'query parameters',
): string | undefined {
  return validateWebsocketDslQueryParams(value, {
    providerLabel: 'Custom TTS',
    definitionLabel,
    supportedVariables: CUSTOM_TTS_DSL_VARIABLES,
    validationContext: VALIDATION_CONTEXT,
  });
}

export function validateCustomTtsResponseParser(
  value: string,
): string | undefined {
  return validateWebsocketDslResponseParser(value, RESPONSE_VALIDATION_OPTIONS);
}

export function parseCustomTtsResponseParser(
  value: string,
): CustomTtsResponseParserRule[] {
  return parseWebsocketDslResponseParser(
    value,
    RESPONSE_VALIDATION_OPTIONS,
  ) as CustomTtsResponseParserRule[];
}

export function parseCustomTtsResponseFrame(
  frame: string | ArrayBuffer | Uint8Array,
  parser: CustomTtsResponseParserRule[],
): CustomTtsParsedResponseFrame {
  const parsed = parseWebsocketDslResponseFrame(frame, parser, RESPONSE_PARSE_OPTIONS);
  if (!parsed.emitted) {
    return {
      kind: 'message',
      payload: parsed.payload,
    };
  }

  return classifyEmittedResponse(parsed.payload, parsed.emitted);
}
