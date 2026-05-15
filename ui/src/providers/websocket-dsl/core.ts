export const WEBSOCKET_DSL_CAST_VALUES = [
  'string',
  'number',
  'boolean',
] as const;

export type WebsocketDslCastTarget = (typeof WEBSOCKET_DSL_CAST_VALUES)[number];
export type WebsocketDslPrimitive = string | number | boolean | null;
export type WebsocketDslJsonValue =
  | WebsocketDslPrimitive
  | WebsocketDslJsonValue[]
  | { [key: string]: WebsocketDslJsonValue };
export type WebsocketDslResponseFrameType = 'binary' | 'json' | 'text';

export type WebsocketDslRequestContext<VariableName extends string = string> =
  Record<VariableName, string>;

export interface WebsocketDslResponseParserRule<
  FrameType extends WebsocketDslResponseFrameType = WebsocketDslResponseFrameType,
  EmitKey extends string = string,
> {
  when: {
    frame: FrameType;
    path?: string;
    equals?: WebsocketDslPrimitive;
  };
  emit: Partial<Record<EmitKey, WebsocketDslJsonValue>>;
}

export interface ParsedWebsocketDslResponseFrame<
  FrameType extends WebsocketDslResponseFrameType = WebsocketDslResponseFrameType,
> {
  frameType?: FrameType;
  payload: unknown;
  emitted?: Record<string, unknown>;
}

interface RequestValidationOptions<VariableName extends string> {
  providerLabel: string;
  definitionLabel: string;
  supportedVariables: readonly VariableName[];
  validationContext: WebsocketDslRequestContext<VariableName>;
}

interface ResponseValidationOptions<
  FrameType extends WebsocketDslResponseFrameType,
  EmitKey extends string,
> {
  providerLabel: string;
  jsonErrorLabel: string;
  supportedFrameTypes: readonly FrameType[];
  supportedEmitKeys: readonly EmitKey[];
  allowedFrameExpressions: readonly WebsocketDslResponseFrameType[];
  allowDecodeBase64?: boolean;
}

interface ResponseParseOptions<
  FrameType extends WebsocketDslResponseFrameType,
> {
  providerLabel: string;
  supportedFrameTypes: readonly FrameType[];
  allowedFrameExpressions: readonly WebsocketDslResponseFrameType[];
  allowDecodeBase64?: boolean;
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && !Array.isArray(value) && typeof value === 'object';
}

function isPrimitive(value: unknown): value is WebsocketDslPrimitive {
  return (
    value === null ||
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean'
  );
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
  const bufferCtor = (globalThis as { Buffer?: { from?: (...args: any[]) => Uint8Array } }).Buffer;
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
  castTarget: WebsocketDslCastTarget,
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

function isOperatorObject(
  value: Record<string, unknown>,
  operator: '$var' | '$path' | '$cast' | '$decode' | '$frame',
): boolean {
  return operator in value;
}

function resolveDefinitionLabel(label: string): string {
  return label.trim().toLowerCase();
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

function isSupportedCastTarget(value: string): value is WebsocketDslCastTarget {
  return WEBSOCKET_DSL_CAST_VALUES.includes(value as WebsocketDslCastTarget);
}

function formatQuotedChoices(values: readonly string[]): string {
  if (values.length === 1) return `"${values[0]}"`;
  if (values.length === 2) return `"${values[0]}" or "${values[1]}"`;
  return `${values
    .slice(0, -1)
    .map(item => `"${item}"`)
    .join(', ')}, or "${values[values.length - 1]}"`;
}

function formatFrameExpressionExamples(
  values: readonly WebsocketDslResponseFrameType[],
): string {
  return values
    .map(item => `{"$frame":"${item}"}`)
    .join(values.length > 1 ? ' or ' : '');
}

function evaluateRequestExpression<VariableName extends string>(
  value: unknown,
  context: WebsocketDslRequestContext<VariableName>,
): unknown {
  if (Array.isArray(value)) {
    return value.map(item => evaluateRequestExpression(item, context));
  }

  if (!isPlainObject(value)) {
    return value;
  }

  if (isOperatorObject(value, '$var')) {
    return context[String(value.$var) as VariableName] ?? '';
  }

  if (isOperatorObject(value, '$cast')) {
    return castPrimitiveValue(
      evaluateRequestExpression(value.value, context),
      String(value.$cast) as WebsocketDslCastTarget,
    );
  }

  return Object.fromEntries(
    Object.entries(value).map(([key, childValue]) => [
      key,
      evaluateRequestExpression(childValue, context),
    ]),
  );
}

function validateRequestExpression<VariableName extends string>(
  value: unknown,
  options: Pick<
    RequestValidationOptions<VariableName>,
    'providerLabel' | 'definitionLabel' | 'supportedVariables'
  >,
): string | undefined {
  if (Array.isArray(value)) {
    for (const item of value) {
      const error = validateRequestExpression(item, options);
      if (error) return error;
    }
    return undefined;
  }

  if (!isPlainObject(value)) {
    return undefined;
  }

  if (isOperatorObject(value, '$var')) {
    if (Object.keys(value).length !== 1 || typeof value.$var !== 'string') {
      return `${options.providerLabel} ${options.definitionLabel} must define "$var" expressions as {"$var":"name"}.`;
    }
    if (
      !options.supportedVariables.includes(value.$var as VariableName)
    ) {
      return `Unsupported ${options.providerLabel.toLowerCase()} variable "${value.$var}" in ${options.definitionLabel}. Supported variables: ${options.supportedVariables.join(', ')}.`;
    }
    return undefined;
  }

  if (isOperatorObject(value, '$cast')) {
    if (
      Object.keys(value).length !== 2 ||
      !('value' in value) ||
      typeof value.$cast !== 'string'
    ) {
      return `${options.providerLabel} ${options.definitionLabel} must define "$cast" expressions as {"$cast":"number","value":...}.`;
    }
    if (!isSupportedCastTarget(value.$cast)) {
      return `${options.providerLabel} ${options.definitionLabel} only supports "$cast" values of ${WEBSOCKET_DSL_CAST_VALUES.join(', ')}.`;
    }
    return validateRequestExpression(value.value, options);
  }

  for (const [key, childValue] of Object.entries(value)) {
    if (key.startsWith('$')) {
      return `Unsupported operator "${key}" in ${options.definitionLabel}.`;
    }
    const error = validateRequestExpression(childValue, options);
    if (error) return error;
  }

  return undefined;
}

interface ResponseExpressionOptions {
  providerLabel: string;
  allowedFrameExpressions: readonly WebsocketDslResponseFrameType[];
  allowDecodeBase64?: boolean;
}

function evaluateResponseExpression(
  value: unknown,
  payload: unknown,
  options: ResponseExpressionOptions,
  rawFrames: Partial<Record<WebsocketDslResponseFrameType, unknown>>,
): unknown {
  if (Array.isArray(value)) {
    return value.map(item =>
      evaluateResponseExpression(item, payload, options, rawFrames),
    );
  }

  if (!isPlainObject(value)) {
    return value;
  }

  if (isOperatorObject(value, '$path')) {
    return getValueAtPath(payload, String(value.$path));
  }

  if (isOperatorObject(value, '$frame')) {
    const frameType = String(value.$frame) as WebsocketDslResponseFrameType;
    if (!options.allowedFrameExpressions.includes(frameType)) {
      return undefined;
    }
    return rawFrames[frameType];
  }

  if (isOperatorObject(value, '$decode')) {
    const decodedValue = evaluateResponseExpression(
      value.value,
      payload,
      options,
      rawFrames,
    );
    if (typeof decodedValue !== 'string') {
      throw new Error(
        `${options.providerLabel} response parser can only decode string values.`,
      );
    }
    if (!options.allowDecodeBase64 || value.$decode !== 'base64') {
      throw new Error(
        `${options.providerLabel} response parser only supports "$decode": "base64".`,
      );
    }
    return decodeBase64ToBytes(decodedValue);
  }

  if (isOperatorObject(value, '$cast')) {
    return castPrimitiveValue(
      evaluateResponseExpression(value.value, payload, options, rawFrames),
      String(value.$cast) as WebsocketDslCastTarget,
    );
  }

  return Object.fromEntries(
    Object.entries(value).map(([key, childValue]) => [
      key,
      evaluateResponseExpression(childValue, payload, options, rawFrames),
    ]),
  );
}

function validateResponseExpression(
  value: unknown,
  options: ResponseExpressionOptions,
): string | undefined {
  if (Array.isArray(value)) {
    for (const item of value) {
      const error = validateResponseExpression(item, options);
      if (error) return error;
    }
    return undefined;
  }

  if (!isPlainObject(value)) {
    return undefined;
  }

  if (isOperatorObject(value, '$path')) {
    if (Object.keys(value).length !== 1 || typeof value.$path !== 'string') {
      return `${options.providerLabel} response parser "$path" expressions must be shaped as {"$path":"field.path"}.`;
    }
    if (!value.$path.trim()) {
      return `${options.providerLabel} response parser "$path" expressions require a non-empty path.`;
    }
    return undefined;
  }

  if (isOperatorObject(value, '$frame')) {
    const allowedFrameExamples = formatFrameExpressionExamples(
      options.allowedFrameExpressions,
    );
    if (
      Object.keys(value).length !== 1 ||
      typeof value.$frame !== 'string' ||
      !options.allowedFrameExpressions.includes(
        value.$frame as WebsocketDslResponseFrameType,
      )
    ) {
      return `${options.providerLabel} response parser "$frame" expressions only support ${allowedFrameExamples}.`;
    }
    return undefined;
  }

  if (isOperatorObject(value, '$decode')) {
    if (!options.allowDecodeBase64) {
      return `${options.providerLabel} response parser does not support "$decode".`;
    }
    if (
      Object.keys(value).length !== 2 ||
      !('value' in value) ||
      value.$decode !== 'base64'
    ) {
      return `${options.providerLabel} response parser "$decode" expressions must be shaped as {"$decode":"base64","value":...}.`;
    }
    return validateResponseExpression(value.value, options);
  }

  if (isOperatorObject(value, '$cast')) {
    if (
      Object.keys(value).length !== 2 ||
      !('value' in value) ||
      typeof value.$cast !== 'string'
    ) {
      return `${options.providerLabel} response parser "$cast" expressions must be shaped as {"$cast":"number","value":...}.`;
    }
    if (!isSupportedCastTarget(value.$cast)) {
      return `${options.providerLabel} response parser only supports "$cast" values of ${WEBSOCKET_DSL_CAST_VALUES.join(', ')}.`;
    }
    return validateResponseExpression(value.value, options);
  }

  for (const [key, childValue] of Object.entries(value)) {
    if (key.startsWith('$')) {
      return `Unsupported operator "${key}" in ${options.providerLabel.toLowerCase()} response parser.`;
    }
    const error = validateResponseExpression(childValue, options);
    if (error) return error;
  }

  return undefined;
}

function matchesResponseRule<
  FrameType extends WebsocketDslResponseFrameType,
  EmitKey extends string,
>(
  rule: WebsocketDslResponseParserRule<FrameType, EmitKey>,
  frameType: FrameType,
  payload: unknown,
): boolean {
  if (rule.when.frame !== frameType) {
    return false;
  }

  if (frameType === 'binary') {
    return true;
  }

  if (frameType === 'text') {
    if (!('equals' in rule.when)) {
      return true;
    }
    return payload === rule.when.equals;
  }

  if (!rule.when.path) {
    return true;
  }

  return getValueAtPath(payload, rule.when.path) === rule.when.equals;
}

export function renderWebsocketDslRequestDefinition<VariableName extends string>(
  definition: string,
  context: WebsocketDslRequestContext<VariableName>,
): unknown {
  const parsed = JSON.parse(definition);
  return evaluateRequestExpression(parsed, context);
}

export function renderWebsocketDslQueryParams<VariableName extends string>(
  definition: string,
  context: WebsocketDslRequestContext<VariableName>,
  providerLabel: string,
): Record<string, string | number | boolean> {
  const rendered = renderWebsocketDslRequestDefinition(definition, context);
  if (!isPlainObject(rendered)) {
    throw new Error(`${providerLabel} query params must resolve to an object.`);
  }

  const entries = Object.entries(rendered).map(([key, value]) => {
    if (
      typeof value !== 'string' &&
      typeof value !== 'number' &&
      typeof value !== 'boolean'
    ) {
      throw new Error(
        `${providerLabel} query param "${key}" must resolve to a string, number, or boolean.`,
      );
    }

    return [key, value];
  });

  return Object.fromEntries(entries);
}

export function validateWebsocketDslRequestDefinition<
  VariableName extends string,
>(
  value: string,
  options: RequestValidationOptions<VariableName>,
): string | undefined {
  const definitionLabel = resolveDefinitionLabel(options.definitionLabel);
  const parsed = parseJsonWithMessage(
    value,
    `Please provide a valid JSON definition for ${definitionLabel}.`,
  );
  if (typeof parsed === 'string') return parsed;
  if (!isPlainObject(parsed)) {
    return `${options.providerLabel} ${definitionLabel} must be a JSON object.`;
  }

  const error = validateRequestExpression(parsed, {
    providerLabel: options.providerLabel,
    definitionLabel,
    supportedVariables: options.supportedVariables,
  });
  if (error) return error;

  try {
    const rendered = renderWebsocketDslRequestDefinition(
      value,
      options.validationContext,
    );
    if (!isPlainObject(rendered)) {
      return `${options.providerLabel} ${definitionLabel} must resolve to a JSON object.`;
    }
    return undefined;
  } catch {
    return `Please provide a valid JSON definition for ${definitionLabel}.`;
  }
}

export function validateWebsocketDslQueryParams<VariableName extends string>(
  value: string,
  options: RequestValidationOptions<VariableName>,
): string | undefined {
  const definitionLabel = resolveDefinitionLabel(options.definitionLabel);
  const requestError = validateWebsocketDslRequestDefinition(value, {
    ...options,
    definitionLabel,
  });
  if (requestError) return requestError;

  try {
    renderWebsocketDslQueryParams(
      value,
      options.validationContext,
      options.providerLabel,
    );
    return undefined;
  } catch {
    return `${options.providerLabel} ${definitionLabel} values must resolve to strings, numbers, or booleans.`;
  }
}

export function validateWebsocketDslResponseParser<
  FrameType extends WebsocketDslResponseFrameType,
  EmitKey extends string,
>(
  value: string,
  options: ResponseValidationOptions<FrameType, EmitKey>,
): string | undefined {
  const parsed = parseJsonWithMessage(
    value,
    `Please provide a valid JSON response parser for ${options.jsonErrorLabel}.`,
  );
  if (typeof parsed === 'string') return parsed;

  if (!Array.isArray(parsed) || parsed.length === 0) {
    return `${options.providerLabel} response parser must be a non-empty JSON array of rules.`;
  }

  for (const [index, rule] of parsed.entries()) {
    if (
      !isPlainObject(rule) ||
      !isPlainObject(rule.when) ||
      !isPlainObject(rule.emit)
    ) {
      return `${options.providerLabel} response parser rule ${index + 1} must define "when" and "emit" objects.`;
    }

    const frame = String(rule.when.frame || '') as FrameType;
    if (!options.supportedFrameTypes.includes(frame)) {
      return `${options.providerLabel} response parser rule ${index + 1} must define when.frame as ${formatQuotedChoices(options.supportedFrameTypes)}.`;
    }

    const hasPath = 'path' in rule.when;
    const hasEquals = 'equals' in rule.when;

    if (frame === 'binary') {
      if (hasPath || hasEquals) {
        return `${options.providerLabel} response parser rule ${index + 1} cannot use when.path or when.equals with binary frames.`;
      }
    } else if (frame === 'json') {
      if (hasPath !== hasEquals) {
        return `${options.providerLabel} response parser rule ${index + 1} must define both when.path and when.equals together.`;
      }
      if (
        hasPath &&
        (typeof rule.when.path !== 'string' || !rule.when.path.trim())
      ) {
        return `${options.providerLabel} response parser rule ${index + 1} must define when.path as a non-empty string.`;
      }
      if (hasEquals && !isPrimitive(rule.when.equals)) {
        return `${options.providerLabel} response parser rule ${index + 1} must define when.equals as a primitive JSON value.`;
      }
    } else if (frame === 'text') {
      if (hasPath) {
        return `${options.providerLabel} response parser rule ${index + 1} cannot use when.path with text frames.`;
      }
      if (hasEquals && !isPrimitive(rule.when.equals)) {
        return `${options.providerLabel} response parser rule ${index + 1} must define when.equals as a primitive JSON value.`;
      }
    }

    const emitKeys = Object.keys(rule.emit);
    if (emitKeys.length === 0) {
      return `${options.providerLabel} response parser rule ${index + 1} must emit at least one value.`;
    }

    for (const key of emitKeys) {
      if (
        !options.supportedEmitKeys.includes(key as EmitKey)
      ) {
        return `${options.providerLabel} response parser rule ${index + 1} cannot emit "${key}". Supported keys: ${options.supportedEmitKeys.join(', ')}.`;
      }

      const expressionError = validateResponseExpression(rule.emit[key], {
        providerLabel: options.providerLabel,
        allowedFrameExpressions: options.allowedFrameExpressions,
        allowDecodeBase64: options.allowDecodeBase64,
      });
      if (expressionError) return expressionError;
    }
  }

  return undefined;
}

export function parseWebsocketDslResponseParser<
  FrameType extends WebsocketDslResponseFrameType,
  EmitKey extends string,
>(
  value: string,
  options: ResponseValidationOptions<FrameType, EmitKey>,
): WebsocketDslResponseParserRule<FrameType, EmitKey>[] {
  const error = validateWebsocketDslResponseParser(value, options);
  if (error) {
    throw new Error(error);
  }

  return JSON.parse(value) as WebsocketDslResponseParserRule<FrameType, EmitKey>[];
}

export function parseWebsocketDslResponseFrame<
  FrameType extends WebsocketDslResponseFrameType,
  EmitKey extends string,
>(
  frame: string | ArrayBuffer | Uint8Array,
  parser: WebsocketDslResponseParserRule<FrameType, EmitKey>[],
  options: ResponseParseOptions<FrameType>,
): ParsedWebsocketDslResponseFrame<FrameType> {
  if (typeof frame !== 'string') {
    const payload = normalizeBinaryFrame(frame);
    const frameType = options.supportedFrameTypes.includes('binary' as FrameType)
      ? ('binary' as FrameType)
      : undefined;
    if (!frameType) {
      return { payload };
    }

    const rule = parser.find(item => matchesResponseRule(item, frameType, payload));
    if (!rule) {
      return {
        frameType,
        payload,
      };
    }

    const emitted = Object.fromEntries(
      Object.entries(rule.emit).map(([key, value]) => [
        key,
        evaluateResponseExpression(
          value,
          payload,
          {
            providerLabel: options.providerLabel,
            allowedFrameExpressions: options.allowedFrameExpressions,
            allowDecodeBase64: options.allowDecodeBase64,
          },
          { binary: payload },
        ),
      ]),
    );

    return {
      frameType,
      payload,
      emitted,
    };
  }

  let payload: unknown = frame;
  let frameType: FrameType | undefined;

  try {
    const parsed = JSON.parse(frame);
    if (options.supportedFrameTypes.includes('json' as FrameType)) {
      payload = parsed;
      frameType = 'json' as FrameType;
    } else if (options.supportedFrameTypes.includes('text' as FrameType)) {
      frameType = 'text' as FrameType;
    }
  } catch {
    if (options.supportedFrameTypes.includes('text' as FrameType)) {
      frameType = 'text' as FrameType;
    }
  }

  if (!frameType) {
    return { payload };
  }

  const rule = parser.find(item => matchesResponseRule(item, frameType, payload));
  if (!rule) {
    return {
      frameType,
      payload,
    };
  }

  const emitted = Object.fromEntries(
    Object.entries(rule.emit).map(([key, value]) => [
      key,
      evaluateResponseExpression(
        value,
        payload,
        {
          providerLabel: options.providerLabel,
          allowedFrameExpressions: options.allowedFrameExpressions,
          allowDecodeBase64: options.allowDecodeBase64,
        },
        frameType === 'text' ? { text: frame } : {},
      ),
    ]),
  );

  return {
    frameType,
    payload,
    emitted,
  };
}
