import { MetadataLike } from '../config-loader';

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

export type CustomTtsDslVariable = (typeof CUSTOM_TTS_DSL_VARIABLES)[number];
export type CustomTtsFlowMode = 'one-shot' | 'two-step';
export type CustomTtsDslCastTarget = 'string' | 'number' | 'boolean';
export type CustomTtsResponseFrameType = 'binary' | 'json';
export type CustomTtsPrimitive = string | number | boolean | null;
export type CustomTtsJsonValue =
  | CustomTtsPrimitive
  | CustomTtsJsonValue[]
  | { [key: string]: CustomTtsJsonValue };

export interface CustomTtsRequestContext {
  text: string;
  message_id: string;
  voice_id: string;
  model: string;
  language: string;
  encoding: string;
  sample_rate: string;
}

export interface CustomTtsResponseParserRule {
  when: {
    frame: CustomTtsResponseFrameType;
    path?: string;
    equals?: CustomTtsPrimitive;
  };
  emit: Partial<
    Record<'audio' | 'message_id' | 'done' | 'error', CustomTtsJsonValue>
  >;
}

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

const SUPPORTED_CAST_TARGETS = ['string', 'number', 'boolean'] as const;
const SUPPORTED_RESPONSE_FRAME_TYPES = ['binary', 'json'] as const;
const SUPPORTED_RESPONSE_EMIT_KEYS = [
  'audio',
  'message_id',
  'done',
  'error',
] as const;
const SUPPORTED_VARIABLE_LABELS = CUSTOM_TTS_DSL_VARIABLES.join(', ');
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

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && !Array.isArray(value) && typeof value === 'object';
}

function isPrimitive(value: unknown): value is CustomTtsPrimitive {
  return (
    value === null ||
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean'
  );
}

function isSupportedDslVariable(
  value: string,
): value is CustomTtsDslVariable {
  return CUSTOM_TTS_DSL_VARIABLES.includes(value as CustomTtsDslVariable);
}

function isSupportedCastTarget(
  value: string,
): value is CustomTtsDslCastTarget {
  return SUPPORTED_CAST_TARGETS.includes(value as CustomTtsDslCastTarget);
}

function isSupportedResponseFrameType(
  value: string,
): value is CustomTtsResponseFrameType {
  return SUPPORTED_RESPONSE_FRAME_TYPES.includes(
    value as CustomTtsResponseFrameType,
  );
}

function isOperatorObject(
  value: Record<string, unknown>,
  operator: '$var' | '$path' | '$cast' | '$decode' | '$frame',
): boolean {
  return operator in value;
}

function getValueAtPath(payload: unknown, path: string): unknown {
  return path
    .split('.')
    .filter(Boolean)
    .reduce<unknown>((current, segment) => {
      if (current === undefined || current === null) {
        return undefined;
      }
      if (Array.isArray(current) && /^\d+$/.test(segment)) {
        return current[Number(segment)];
      }
      if (typeof current === 'object') {
        return (current as Record<string, unknown>)[segment];
      }
      return undefined;
    }, payload);
}

function normalizeBinaryFrame(frame: ArrayBuffer | Uint8Array): Uint8Array {
  return frame instanceof Uint8Array ? frame : new Uint8Array(frame);
}

function decodeBase64ToBytes(value: string): Uint8Array {
  const bufferCtor = (globalThis as { Buffer?: any }).Buffer;
  if (bufferCtor?.from) {
    return Uint8Array.from(bufferCtor.from(value, 'base64'));
  }

  if (typeof globalThis.atob === 'function') {
    const decoded = globalThis.atob(value);
    const bytes = new Uint8Array(decoded.length);
    for (let index = 0; index < decoded.length; index += 1) {
      bytes[index] = decoded.charCodeAt(index);
    }
    return bytes;
  }

  throw new Error('Base64 decoding is not available in this environment.');
}

function castPrimitiveValue(
  value: unknown,
  castTarget: CustomTtsDslCastTarget,
): string | number | boolean {
  switch (castTarget) {
    case 'string':
      return value === null || value === undefined ? '' : String(value);
    case 'number': {
      const parsed = Number(value);
      if (Number.isNaN(parsed)) {
        throw new Error(`Cannot cast value "${String(value)}" to number.`);
      }
      return parsed;
    }
    case 'boolean':
      if (typeof value === 'boolean') return value;
      if (typeof value === 'number') return value !== 0;
      if (typeof value === 'string') {
        if (value === 'true') return true;
        if (value === 'false') return false;
      }
      throw new Error(`Cannot cast value "${String(value)}" to boolean.`);
  }
}

function evaluateRequestExpression(
  value: unknown,
  context: CustomTtsRequestContext,
): unknown {
  if (Array.isArray(value)) {
    return value.map(item => evaluateRequestExpression(item, context));
  }

  if (!isPlainObject(value)) {
    return value;
  }

  if (isOperatorObject(value, '$var')) {
    return context[String(value.$var) as CustomTtsDslVariable] ?? '';
  }

  if (isOperatorObject(value, '$cast')) {
    return castPrimitiveValue(
      evaluateRequestExpression(value.value, context),
      String(value.$cast) as CustomTtsDslCastTarget,
    );
  }

  return Object.fromEntries(
    Object.entries(value).map(([key, childValue]) => [
      key,
      evaluateRequestExpression(childValue, context),
    ]),
  );
}

function evaluateResponseExpression(
  value: unknown,
  payload: unknown,
  rawBinary?: Uint8Array,
): unknown {
  if (Array.isArray(value)) {
    return value.map(item => evaluateResponseExpression(item, payload, rawBinary));
  }

  if (!isPlainObject(value)) {
    return value;
  }

  if (isOperatorObject(value, '$path')) {
    return getValueAtPath(payload, String(value.$path));
  }

  if (isOperatorObject(value, '$frame')) {
    if (value.$frame !== 'binary' || !rawBinary) {
      return undefined;
    }
    return rawBinary;
  }

  if (isOperatorObject(value, '$decode')) {
    const decodedValue = evaluateResponseExpression(value.value, payload, rawBinary);
    if (typeof decodedValue !== 'string') {
      throw new Error('Custom TTS response parser can only decode string values.');
    }
    if (value.$decode !== 'base64') {
      throw new Error('Custom TTS response parser only supports "$decode": "base64".');
    }
    return decodeBase64ToBytes(decodedValue);
  }

  if (isOperatorObject(value, '$cast')) {
    return castPrimitiveValue(
      evaluateResponseExpression(value.value, payload, rawBinary),
      String(value.$cast) as CustomTtsDslCastTarget,
    );
  }

  return Object.fromEntries(
    Object.entries(value).map(([key, childValue]) => [
      key,
      evaluateResponseExpression(childValue, payload, rawBinary),
    ]),
  );
}

function validateRequestExpression(
  value: unknown,
  sectionLabel: string,
  path = '$',
): string | undefined {
  if (Array.isArray(value)) {
    for (let index = 0; index < value.length; index += 1) {
      const error = validateRequestExpression(
        value[index],
        sectionLabel,
        `${path}[${index}]`,
      );
      if (error) return error;
    }
    return undefined;
  }

  if (!isPlainObject(value)) {
    return undefined;
  }

  if (isOperatorObject(value, '$var')) {
    if (Object.keys(value).length !== 1 || typeof value.$var !== 'string') {
      return `Custom TTS ${sectionLabel} must define "$var" expressions as {"$var":"name"}.`;
    }
    if (!isSupportedDslVariable(value.$var)) {
      return `Unsupported custom TTS variable "${value.$var}" in ${sectionLabel}. Supported variables: ${SUPPORTED_VARIABLE_LABELS}.`;
    }
    return undefined;
  }

  if (isOperatorObject(value, '$cast')) {
    if (
      Object.keys(value).length !== 2 ||
      !('value' in value) ||
      typeof value.$cast !== 'string'
    ) {
      return `Custom TTS ${sectionLabel} must define "$cast" expressions as {"$cast":"number","value":...}.`;
    }
    if (!isSupportedCastTarget(value.$cast)) {
      return `Custom TTS ${sectionLabel} only supports "$cast" values of ${SUPPORTED_CAST_TARGETS.join(', ')}.`;
    }
    return validateRequestExpression(value.value, sectionLabel, `${path}.value`);
  }

  for (const [key, childValue] of Object.entries(value)) {
    if (key.startsWith('$')) {
      return `Unsupported operator "${key}" in ${sectionLabel}.`;
    }
    const error = validateRequestExpression(
      childValue,
      sectionLabel,
      `${path}.${key}`,
    );
    if (error) return error;
  }

  return undefined;
}

function validateResponseExpression(
  value: unknown,
  path = '$',
): string | undefined {
  if (Array.isArray(value)) {
    for (let index = 0; index < value.length; index += 1) {
      const error = validateResponseExpression(value[index], `${path}[${index}]`);
      if (error) return error;
    }
    return undefined;
  }

  if (!isPlainObject(value)) {
    return undefined;
  }

  if (isOperatorObject(value, '$path')) {
    if (Object.keys(value).length !== 1 || typeof value.$path !== 'string') {
      return 'Custom TTS response parser "$path" expressions must be shaped as {"$path":"field.path"}.';
    }
    if (!value.$path.trim()) {
      return 'Custom TTS response parser "$path" expressions require a non-empty path.';
    }
    return undefined;
  }

  if (isOperatorObject(value, '$frame')) {
    if (Object.keys(value).length !== 1 || value.$frame !== 'binary') {
      return 'Custom TTS response parser "$frame" expressions only support {"$frame":"binary"}.';
    }
    return undefined;
  }

  if (isOperatorObject(value, '$decode')) {
    if (
      Object.keys(value).length !== 2 ||
      !('value' in value) ||
      value.$decode !== 'base64'
    ) {
      return 'Custom TTS response parser "$decode" expressions must be shaped as {"$decode":"base64","value":...}.';
    }
    return validateResponseExpression(value.value, `${path}.value`);
  }

  if (isOperatorObject(value, '$cast')) {
    if (
      Object.keys(value).length !== 2 ||
      !('value' in value) ||
      typeof value.$cast !== 'string'
    ) {
      return 'Custom TTS response parser "$cast" expressions must be shaped as {"$cast":"number","value":...}.';
    }
    if (!isSupportedCastTarget(value.$cast)) {
      return `Custom TTS response parser only supports "$cast" values of ${SUPPORTED_CAST_TARGETS.join(', ')}.`;
    }
    return validateResponseExpression(value.value, `${path}.value`);
  }

  for (const [key, childValue] of Object.entries(value)) {
    if (key.startsWith('$')) {
      return `Unsupported operator "${key}" in custom TTS response parser.`;
    }
    const error = validateResponseExpression(childValue, `${path}.${key}`);
    if (error) return error;
  }

  return undefined;
}

function parseJsonWithMessage(
  value: string,
  errorMessage: string,
): unknown | string {
  try {
    return JSON.parse(value);
  } catch {
    return errorMessage;
  }
}

function resolveSectionLabel(label: string): string {
  return label.trim().toLowerCase();
}

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

function matchesResponseRule(
  rule: CustomTtsResponseParserRule,
  frameType: CustomTtsResponseFrameType,
  payload: unknown,
): boolean {
  if (rule.when.frame !== frameType) {
    return false;
  }

  if (!rule.when.path) {
    return true;
  }

  return getValueAtPath(payload, rule.when.path) === rule.when.equals;
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
  const parsed = JSON.parse(definition);
  return evaluateRequestExpression(parsed, context);
}

export function renderCustomTtsQueryParams(
  definition: string,
  context: CustomTtsRequestContext,
): Record<string, string | number | boolean> {
  const rendered = renderCustomTtsRequestDefinition(definition, context);
  if (!isPlainObject(rendered)) {
    throw new Error('Custom TTS query params must resolve to an object.');
  }

  const entries = Object.entries(rendered).map(([key, value]) => {
    if (
      typeof value !== 'string' &&
      typeof value !== 'number' &&
      typeof value !== 'boolean'
    ) {
      throw new Error(
        `Custom TTS query param "${key}" must resolve to a string, number, or boolean.`,
      );
    }
    return [key, value];
  });

  return Object.fromEntries(entries);
}

export function validateCustomTtsRequestDefinition(
  value: string,
  definitionLabel = 'request definition',
): string | undefined {
  const normalizedLabel = resolveSectionLabel(definitionLabel);
  const parsed = parseJsonWithMessage(
    value,
    `Please provide a valid JSON definition for ${normalizedLabel}.`,
  );
  if (typeof parsed === 'string') return parsed;
  if (!isPlainObject(parsed)) {
    return `Custom TTS ${normalizedLabel} must be a JSON object.`;
  }

  const error = validateRequestExpression(parsed, normalizedLabel);
  if (error) return error;

  try {
    const rendered = renderCustomTtsRequestDefinition(value, VALIDATION_CONTEXT);
    if (!isPlainObject(rendered)) {
      return `Custom TTS ${normalizedLabel} must resolve to a JSON object.`;
    }
    return undefined;
  } catch {
    return `Please provide a valid JSON definition for ${normalizedLabel}.`;
  }
}

export function validateCustomTtsQueryParams(
  value: string,
  definitionLabel = 'query parameters',
): string | undefined {
  const normalizedLabel = resolveSectionLabel(definitionLabel);
  const requestError = validateCustomTtsRequestDefinition(value, normalizedLabel);
  if (requestError) return requestError;

  try {
    renderCustomTtsQueryParams(value, VALIDATION_CONTEXT);
    return undefined;
  } catch {
    return `Custom TTS ${normalizedLabel} values must resolve to strings, numbers, or booleans.`;
  }
}

export function validateCustomTtsResponseParser(
  value: string,
): string | undefined {
  const parsed = parseJsonWithMessage(
    value,
    'Please provide a valid JSON response parser for custom TTS.',
  );
  if (typeof parsed === 'string') return parsed;

  if (!Array.isArray(parsed) || parsed.length === 0) {
    return 'Custom TTS response parser must be a non-empty JSON array of rules.';
  }

  for (const [index, rule] of parsed.entries()) {
    if (!isPlainObject(rule) || !isPlainObject(rule.when) || !isPlainObject(rule.emit)) {
      return `Custom TTS response parser rule ${index + 1} must define "when" and "emit" objects.`;
    }

    const frame = String(rule.when.frame || '');
    if (!isSupportedResponseFrameType(frame)) {
      return `Custom TTS response parser rule ${index + 1} must define when.frame as "binary" or "json".`;
    }

    if (frame === 'binary' && ('path' in rule.when || 'equals' in rule.when)) {
      return `Custom TTS response parser rule ${index + 1} cannot use when.path or when.equals with binary frames.`;
    }

    if (frame === 'json') {
      const hasPath = 'path' in rule.when;
      const hasEquals = 'equals' in rule.when;
      if (hasPath !== hasEquals) {
        return `Custom TTS response parser rule ${index + 1} must define both when.path and when.equals together.`;
      }
      if (
        hasPath &&
        (typeof rule.when.path !== 'string' || !rule.when.path.trim())
      ) {
        return `Custom TTS response parser rule ${index + 1} must define when.path as a non-empty string.`;
      }
      if (hasEquals && !isPrimitive(rule.when.equals)) {
        return `Custom TTS response parser rule ${index + 1} must define when.equals as a primitive JSON value.`;
      }
    }

    const emitKeys = Object.keys(rule.emit);
    if (emitKeys.length === 0) {
      return `Custom TTS response parser rule ${index + 1} must emit at least one value.`;
    }

    for (const key of emitKeys) {
      if (
        !SUPPORTED_RESPONSE_EMIT_KEYS.includes(
          key as (typeof SUPPORTED_RESPONSE_EMIT_KEYS)[number],
        )
      ) {
        return `Custom TTS response parser rule ${index + 1} cannot emit "${key}". Supported keys: ${SUPPORTED_RESPONSE_EMIT_KEYS.join(', ')}.`;
      }
      const expressionError = validateResponseExpression(rule.emit[key]);
      if (expressionError) return expressionError;
    }
  }

  return undefined;
}

export function parseCustomTtsResponseParser(
  value: string,
): CustomTtsResponseParserRule[] {
  const error = validateCustomTtsResponseParser(value);
  if (error) {
    throw new Error(error);
  }

  return JSON.parse(value) as CustomTtsResponseParserRule[];
}

export function parseCustomTtsResponseFrame(
  frame: string | ArrayBuffer | Uint8Array,
  parser: CustomTtsResponseParserRule[],
): CustomTtsParsedResponseFrame {
  if (typeof frame !== 'string') {
    const rawBinary = normalizeBinaryFrame(frame);
    const rule = parser.find(item => matchesResponseRule(item, 'binary', rawBinary));
    if (!rule) {
      return {
        kind: 'message',
        payload: rawBinary,
      };
    }

    const emitted = Object.fromEntries(
      Object.entries(rule.emit).map(([key, value]) => [
        key,
        evaluateResponseExpression(value, rawBinary, rawBinary),
      ]),
    );
    return classifyEmittedResponse(rawBinary, emitted);
  }

  let payload: unknown;
  try {
    payload = JSON.parse(frame);
  } catch {
    return {
      kind: 'message',
      payload: frame,
    };
  }

  const rule = parser.find(item => matchesResponseRule(item, 'json', payload));
  if (!rule) {
    return {
      kind: 'message',
      payload,
    };
  }

  const emitted = Object.fromEntries(
    Object.entries(rule.emit).map(([key, value]) => [
      key,
      evaluateResponseExpression(value, payload),
    ]),
  );
  return classifyEmittedResponse(payload, emitted);
}
