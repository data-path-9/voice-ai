import { MetadataLike } from '../config-loader';
import {
  WebsocketDslCastTarget,
  WebsocketDslJsonValue,
  WebsocketDslPrimitive,
  WebsocketDslRequestFrameType,
  WebsocketDslRequestRule,
  WebsocketDslResponseRule,
  parseWebsocketDslRequestRules,
  parseWebsocketDslResponseFrame,
  parseWebsocketDslResponseRules,
  renderWebsocketDslQueryParams,
  renderWebsocketDslScopedValue,
  validateWebsocketDslQueryParams,
  validateWebsocketDslRequestRules,
  validateWebsocketDslResponseRules,
} from '../websocket-dsl/core';

export const CUSTOM_TTS_QUERY_PARAMS_KEY = 'speak.query_params';
export const CUSTOM_TTS_REQUEST_RULES_KEY = 'speak.request_rules';
export const CUSTOM_TTS_RESPONSE_RULES_KEY = 'speak.response_rules';

export const CUSTOM_TTS_DSL_VARIABLES = [
  'message_id',
  'voice_id',
  'model',
  'language',
  'encoding',
  'sample_rate',
] as const;

export const CUSTOM_TTS_REQUEST_PACKETS = [
  'text',
  'done',
  'interrupt',
] as const;

type CustomTtsResponseEmitKey = 'audio' | 'message_id' | 'done' | 'error';

export type CustomTtsDslVariable = (typeof CUSTOM_TTS_DSL_VARIABLES)[number];
export type CustomTtsRequestPacket =
  (typeof CUSTOM_TTS_REQUEST_PACKETS)[number];
export type CustomTtsFlowMode = 'one-shot' | 'two-step';
export type CustomTtsDslCastTarget = WebsocketDslCastTarget;
export type CustomTtsRequestFrameType = WebsocketDslRequestFrameType;
export type CustomTtsResponseFrameType = 'binary' | 'json';
export type CustomTtsPrimitive = WebsocketDslPrimitive;
export type CustomTtsJsonValue = WebsocketDslJsonValue;

export interface CustomTtsQueryContext {
  text: string;
  message_id: string;
  voice_id: string;
  model: string;
  language: string;
  encoding: string;
  sample_rate: string;
}

export interface CustomTtsRequestRuleContext {
  config: {
    voice: {
      id: string;
    };
    model: string;
    language: string;
    audio: {
      encoding: string;
      sample_rate: string;
    };
  };
  packet: {
    kind: CustomTtsRequestPacket;
    message_id: string;
    text: string;
  };
}

export type CustomTtsRequestRule = WebsocketDslRequestRule<
  CustomTtsRequestPacket,
  CustomTtsRequestFrameType
>;

export type CustomTtsResponseRule = WebsocketDslResponseRule<
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
  definitionLabel: 'response rules',
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

const QUERY_VALIDATION_CONTEXT: CustomTtsQueryContext = {
  text: 'Hello world',
  message_id: 'msg_123',
  voice_id: 'voice_123',
  model: 'model_123',
  language: 'en-US',
  encoding: 'LINEAR16',
  sample_rate: '16000',
};

const REQUEST_RULE_VALIDATION_CONTEXTS: Record<
  CustomTtsRequestPacket,
  CustomTtsRequestRuleContext
> = {
  text: {
    config: {
      voice: { id: 'voice_123' },
      model: 'model_123',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    },
    packet: {
      kind: 'text',
      message_id: 'msg_123',
      text: 'Hello world',
    },
  },
  done: {
    config: {
      voice: { id: 'voice_123' },
      model: 'model_123',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    },
    packet: {
      kind: 'done',
      message_id: 'msg_123',
      text: '',
    },
  },
  interrupt: {
    config: {
      voice: { id: 'voice_123' },
      model: 'model_123',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    },
    packet: {
      kind: 'interrupt',
      message_id: 'msg_123',
      text: '',
    },
  },
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

export const CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE = `[
  {
    "when": { "packet": "text" },
    "send": {
      "frame": "json",
      "body": {
        "text": { "$path": "packet.text" },
        "voice_id": "narrator-1",
        "message_id": { "$path": "packet.message_id" },
        "model": "sonic-2",
        "language": "en-US",
        "audio": {
          "encoding": { "$path": "config.audio.encoding" },
          "sample_rate": {
            "$cast": "number",
            "value": { "$path": "config.audio.sample_rate" }
          }
        }
      }
    }
  }
]`;

export const CUSTOM_TTS_REQUEST_RULES_DONE_EXAMPLE = `[
  {
    "when": { "packet": "text" },
    "send": {
      "frame": "json",
      "body": {
        "text": { "$path": "packet.text" },
        "voice_id": { "$path": "config.voice.id" },
        "message_id": { "$path": "packet.message_id" }
      }
    }
  },
  {
    "when": { "packet": "done" },
    "send": {
      "frame": "json",
      "body": {
        "type": "done",
        "message_id": { "$path": "packet.message_id" }
      }
    }
  }
]`;

export const CUSTOM_TTS_REQUEST_RULES_INTERRUPT_EXAMPLE = `[
  {
    "when": { "packet": "text" },
    "send": {
      "frame": "json",
      "body": {
        "text": { "$path": "packet.text" },
        "voice_id": { "$path": "config.voice.id" },
        "message_id": { "$path": "packet.message_id" }
      }
    }
  },
  {
    "when": { "packet": "interrupt" },
    "send": {
      "frame": "json",
      "body": {
        "type": "interrupt",
        "message_id": { "$path": "packet.message_id" }
      }
    }
  }
]`;

export const CUSTOM_TTS_RESPONSE_RULES_BINARY_EXAMPLE = `[
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

export const CUSTOM_TTS_RESPONSE_RULES_JSON_AUDIO_EXAMPLE = `[
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

function parseRequestRulesForFlow(
  value?: string | null,
): CustomTtsRequestRule[] | null {
  if (!value?.trim()) return null;

  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? (parsed as CustomTtsRequestRule[]) : null;
  } catch {
    return null;
  }
}

export function resolveCustomTtsFlowMode(
  requestRules?: string | null,
): CustomTtsFlowMode {
  const rules = parseRequestRulesForFlow(requestRules);
  if (!rules?.some(rule => rule.when?.packet === 'done')) {
    return 'one-shot';
  }

  return 'two-step';
}

export function getCustomTtsFlowMode(
  metadata: MetadataLike[],
): CustomTtsFlowMode {
  const requestRules = metadata
    .find(item => item.getKey() === CUSTOM_TTS_REQUEST_RULES_KEY)
    ?.getValue();
  return resolveCustomTtsFlowMode(requestRules);
}

export function renderCustomTtsQueryParams(
  definition: string,
  context: CustomTtsQueryContext,
): Record<string, string | number | boolean> {
  return renderWebsocketDslQueryParams(definition, context, 'Custom TTS');
}

export function renderCustomTtsRequestRuleBody(
  rule: CustomTtsRequestRule,
  context: CustomTtsRequestRuleContext,
): unknown {
  return renderWebsocketDslScopedValue(JSON.stringify(rule.send.body), context);
}

export function validateCustomTtsQueryParams(
  value: string,
  definitionLabel = 'query parameters',
): string | undefined {
  return validateWebsocketDslQueryParams(value, {
    providerLabel: 'Custom TTS',
    definitionLabel,
    supportedVariables: CUSTOM_TTS_DSL_VARIABLES,
    validationContext: QUERY_VALIDATION_CONTEXT,
  });
}

export function validateCustomTtsRequestRules(
  value: string,
): string | undefined {
  const error = validateWebsocketDslRequestRules(value, {
    providerLabel: 'Custom TTS',
    definitionLabel: 'request rules',
    jsonErrorLabel: 'custom TTS',
    supportedPackets: CUSTOM_TTS_REQUEST_PACKETS,
    supportedFrameTypes: ['binary', 'json', 'text'] as const,
    supportedPathRoots: ['config', 'packet'] as const,
    validationContexts: REQUEST_RULE_VALIDATION_CONTEXTS,
  });
  if (error) return error;

  try {
    const rules = JSON.parse(value) as CustomTtsRequestRule[];
    if (!rules.some(rule => rule.when?.packet === 'text')) {
      return 'Custom TTS request rules must contain at least one rule with when.packet "text".';
    }
  } catch {
    return 'Please provide valid JSON request rules for custom TTS.';
  }

  return undefined;
}

export function parseCustomTtsRequestRules(
  value: string,
): CustomTtsRequestRule[] {
  const error = validateCustomTtsRequestRules(value);
  if (error) {
    throw new Error(error);
  }

  return parseWebsocketDslRequestRules(value, {
    providerLabel: 'Custom TTS',
    definitionLabel: 'request rules',
    jsonErrorLabel: 'custom TTS',
    supportedPackets: CUSTOM_TTS_REQUEST_PACKETS,
    supportedFrameTypes: ['binary', 'json', 'text'] as const,
    supportedPathRoots: ['config', 'packet'] as const,
    validationContexts: REQUEST_RULE_VALIDATION_CONTEXTS,
  }) as CustomTtsRequestRule[];
}

export function validateCustomTtsResponseRules(
  value: string,
): string | undefined {
  return validateWebsocketDslResponseRules(value, RESPONSE_VALIDATION_OPTIONS);
}

export function parseCustomTtsResponseRules(
  value: string,
): CustomTtsResponseRule[] {
  return parseWebsocketDslResponseRules(
    value,
    RESPONSE_VALIDATION_OPTIONS,
  ) as CustomTtsResponseRule[];
}

export function parseCustomTtsResponseFrame(
  frame: string | ArrayBuffer | Uint8Array,
  rules: CustomTtsResponseRule[],
): CustomTtsParsedResponseFrame {
  const parsed = parseWebsocketDslResponseFrame(
    frame,
    rules,
    RESPONSE_PARSE_OPTIONS,
  );
  if (!parsed.emitted) {
    return {
      kind: 'message',
      payload: parsed.payload,
    };
  }

  return classifyEmittedResponse(parsed.payload, parsed.emitted);
}
