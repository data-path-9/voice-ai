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

export const CUSTOM_STT_QUERY_PARAMS_KEY = 'listen.query_params';
export const CUSTOM_STT_REQUEST_RULES_KEY = 'listen.request_rules';
export const CUSTOM_STT_RESPONSE_RULES_KEY = 'listen.response_rules';

export const CUSTOM_STT_DSL_VARIABLES = [
  'model',
  'language',
  'encoding',
  'sample_rate',
] as const;

export const CUSTOM_STT_REQUEST_PACKETS = [
  'turn_change',
  'audio',
  'interrupt',
] as const;

type CustomSttResponseEmitKey =
  | 'script'
  | 'confidence'
  | 'language'
  | 'interim'
  | 'error';

export type CustomSttDslVariable = (typeof CUSTOM_STT_DSL_VARIABLES)[number];
export type CustomSttRequestPacket =
  (typeof CUSTOM_STT_REQUEST_PACKETS)[number];
export type CustomSttDslCastTarget = WebsocketDslCastTarget;
export type CustomSttRequestFrameType = WebsocketDslRequestFrameType;
export type CustomSttResponseFrameType = 'json' | 'text';
export type CustomSttPrimitive = WebsocketDslPrimitive;
export type CustomSttJsonValue = WebsocketDslJsonValue;

export interface CustomSttQueryContext {
  model: string;
  language: string;
  encoding: string;
  sample_rate: string;
}

export interface CustomSttRequestRuleContext {
  config: {
    model: string;
    language: string;
    audio: {
      encoding: string;
      sample_rate: string;
    };
  };
  packet: {
    kind: CustomSttRequestPacket;
    context_id: string;
    audio?: {
      bytes: string;
      base64: string;
      pcm_base64?: string;
      wav_base64?: string;
    };
  };
}

export type CustomSttRequestRule = WebsocketDslRequestRule<
  CustomSttRequestPacket,
  CustomSttRequestFrameType
>;

export type CustomSttResponseRule = WebsocketDslResponseRule<
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
  definitionLabel: 'response rules',
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

const QUERY_VALIDATION_CONTEXT: CustomSttQueryContext = {
  model: 'nova-3',
  language: 'en-US',
  encoding: 'LINEAR16',
  sample_rate: '16000',
};

const REQUEST_RULE_VALIDATION_CONTEXTS: Record<
  CustomSttRequestPacket,
  CustomSttRequestRuleContext
> = {
  turn_change: {
    config: {
      model: 'nova-3',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    },
    packet: {
      kind: 'turn_change',
      context_id: 'ctx_123',
    },
  },
  audio: {
    config: {
      model: 'nova-3',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    },
    packet: {
      kind: 'audio',
      context_id: 'ctx_123',
      audio: {
        bytes: 'AAEC',
        base64: 'AAEC',
        pcm_base64: 'AAEC',
        wav_base64: 'UklGRg==',
      },
    },
  },
  interrupt: {
    config: {
      model: 'nova-3',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    },
    packet: {
      kind: 'interrupt',
      context_id: 'ctx_123',
    },
  },
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

export const CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE = `[
  {
    "when": { "packet": "audio" },
    "send": {
      "frame": "binary",
      "body": { "$path": "packet.audio.bytes" }
    }
  }
]`;

export const CUSTOM_STT_REQUEST_RULES_JSON_AUDIO_EXAMPLE = `[
  {
    "when": { "packet": "audio" },
    "send": {
      "frame": "json",
      "body": {
        "audio": { "$path": "packet.audio.base64" },
        "encoding": { "$path": "config.audio.encoding" },
        "sample_rate": {
          "$cast": "number",
          "value": { "$path": "config.audio.sample_rate" }
        }
      }
    }
  }
]`;

export const CUSTOM_STT_REQUEST_RULES_TURN_CHANGE_EXAMPLE = `[
  {
    "when": { "packet": "turn_change" },
    "send": {
      "frame": "json",
      "body": {
        "type": "start",
        "language": { "$path": "config.language" },
        "encoding": { "$path": "config.audio.encoding" },
        "sample_rate": {
          "$cast": "number",
          "value": { "$path": "config.audio.sample_rate" }
        }
      }
    }
  },
  {
    "when": { "packet": "audio" },
    "send": {
      "frame": "binary",
      "body": { "$path": "packet.audio.bytes" }
    }
  }
]`;

export const CUSTOM_STT_REQUEST_RULES_INTERRUPT_EXAMPLE = `[
  {
    "when": { "packet": "audio" },
    "send": {
      "frame": "binary",
      "body": { "$path": "packet.audio.bytes" }
    }
  },
  {
    "when": { "packet": "interrupt" },
    "send": {
      "frame": "json",
      "body": { "type": "flush" }
    }
  }
]`;

export const CUSTOM_STT_RESPONSE_RULES_EXAMPLE = `[
  {
    "when": { "frame": "text" },
    "emit": {
      "script": { "$frame": "text" },
      "language": "hi",
      "interim": true
    }
  }
]`;

export const CUSTOM_STT_RESPONSE_RULES_JSON_EXAMPLE = `[
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

export const CUSTOM_STT_RESPONSE_RULES_NESTED_EXAMPLE = `[
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

export function renderCustomSttQueryParams(
  definition: string,
  context: CustomSttQueryContext,
): Record<string, string | number | boolean> {
  return renderWebsocketDslQueryParams(definition, context, 'Custom STT');
}

export function renderCustomSttRequestRuleBody(
  rule: CustomSttRequestRule,
  context: CustomSttRequestRuleContext,
): unknown {
  return renderWebsocketDslScopedValue(JSON.stringify(rule.send.body), context);
}

export function validateCustomSttQueryParams(
  value: string,
  definitionLabel = 'query parameters',
): string | undefined {
  return validateWebsocketDslQueryParams(value, {
    providerLabel: 'Custom STT',
    definitionLabel,
    supportedVariables: CUSTOM_STT_DSL_VARIABLES,
    validationContext: QUERY_VALIDATION_CONTEXT,
  });
}

export function validateCustomSttRequestRules(
  value: string,
): string | undefined {
  const error = validateWebsocketDslRequestRules(value, {
    providerLabel: 'Custom STT',
    definitionLabel: 'request rules',
    jsonErrorLabel: 'custom STT',
    supportedPackets: CUSTOM_STT_REQUEST_PACKETS,
    supportedFrameTypes: ['binary', 'json', 'text'] as const,
    supportedPathRoots: ['config', 'packet'] as const,
    validationContexts: REQUEST_RULE_VALIDATION_CONTEXTS,
  });
  if (error) return error;

  try {
    const rules = JSON.parse(value) as CustomSttRequestRule[];
    if (!rules.some(rule => rule.when?.packet === 'audio')) {
      return 'Custom STT request rules must contain at least one rule with when.packet "audio".';
    }
  } catch {
    return 'Please provide valid JSON request rules for custom STT.';
  }

  return undefined;
}

export function parseCustomSttRequestRules(
  value: string,
): CustomSttRequestRule[] {
  const error = validateCustomSttRequestRules(value);
  if (error) {
    throw new Error(error);
  }

  return parseWebsocketDslRequestRules(value, {
    providerLabel: 'Custom STT',
    definitionLabel: 'request rules',
    jsonErrorLabel: 'custom STT',
    supportedPackets: CUSTOM_STT_REQUEST_PACKETS,
    supportedFrameTypes: ['binary', 'json', 'text'] as const,
    supportedPathRoots: ['config', 'packet'] as const,
    validationContexts: REQUEST_RULE_VALIDATION_CONTEXTS,
  }) as CustomSttRequestRule[];
}

export function validateCustomSttResponseRules(
  value: string,
): string | undefined {
  return validateWebsocketDslResponseRules(value, RESPONSE_VALIDATION_OPTIONS);
}

export function parseCustomSttResponseRules(
  value: string,
): CustomSttResponseRule[] {
  return parseWebsocketDslResponseRules(
    value,
    RESPONSE_VALIDATION_OPTIONS,
  ) as CustomSttResponseRule[];
}

export function parseCustomSttResponseFrame(
  frame: string | ArrayBuffer | Uint8Array,
  rules: CustomSttResponseRule[],
): CustomSttParsedResponseFrame {
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
