import {
  CUSTOM_TTS_DONE_REQUEST_KEY,
  CUSTOM_TTS_DSL_VARIABLES,
  CUSTOM_TTS_QUERY_PARAMS_EXAMPLE,
  CUSTOM_TTS_RESPONSE_PARSER_BINARY_EXAMPLE,
  CUSTOM_TTS_RESPONSE_PARSER_JSON_AUDIO_EXAMPLE,
  CUSTOM_TTS_TEXT_REQUEST_EXAMPLE,
  getCustomTtsFlowMode,
  parseCustomTtsResponseFrame,
  parseCustomTtsResponseParser,
  renderCustomTtsQueryParams,
  renderCustomTtsRequestDefinition,
  resolveCustomTtsFlowMode,
  validateCustomTtsQueryParams,
  validateCustomTtsRequestDefinition,
  validateCustomTtsResponseParser,
} from '../contract';

const metadata = (key: string, value: string) => ({
  getKey: () => key,
  getValue: () => value,
});

describe('custom-tts websocket DSL helpers', () => {
  it('renders request bindings and query params from the shared DSL', () => {
    const renderedRequest = renderCustomTtsRequestDefinition(
      CUSTOM_TTS_TEXT_REQUEST_EXAMPLE,
      {
        text: 'Hello "world"',
        message_id: 'msg-7',
        voice_id: 'voice-9',
        model: 'sonic-2',
        language: 'en-US',
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    );

    expect(renderedRequest).toEqual({
      text: 'Hello "world"',
      voice_id: 'voice-9',
      message_id: 'msg-7',
      model: 'sonic-2',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: 16000,
      },
    });

    expect(
      renderCustomTtsQueryParams(CUSTOM_TTS_QUERY_PARAMS_EXAMPLE, {
        text: 'Hello "world"',
        message_id: 'msg-7',
        voice_id: 'voice-9',
        model: 'sonic-2',
        language: 'en-US',
        encoding: 'LINEAR16',
        sample_rate: '16000',
      }),
    ).toEqual({
      language: 'en-US',
      model: 'sonic-2',
      voice: 'voice-9',
      message_id: 'msg-7',
      sample_rate: 16000,
    });
  });

  it('selects one-shot vs two-step mode from the done request', () => {
    expect(resolveCustomTtsFlowMode('')).toBe('one-shot');
    expect(resolveCustomTtsFlowMode('  ')).toBe('one-shot');
    expect(resolveCustomTtsFlowMode('{"type":"done"}')).toBe('two-step');

    expect(
      getCustomTtsFlowMode([
        metadata(CUSTOM_TTS_DONE_REQUEST_KEY, '{"type":"done"}'),
      ]),
    ).toBe('two-step');
  });

  it('rejects unsupported DSL variables and query param objects that do not resolve to primitives', () => {
    const variableResult = validateCustomTtsRequestDefinition(
      '{"message_id":{"$var":"request_id"}}',
      'Text Request',
    );

    expect(variableResult).toContain('"request_id"');
    for (const variable of CUSTOM_TTS_DSL_VARIABLES) {
      expect(variableResult).toContain(variable);
    }

    expect(
      validateCustomTtsQueryParams(
        '{"audio":{"encoding":{"$var":"encoding"}}}',
        'Query Parameters',
      ),
    ).toBe(
      'Custom TTS query parameters values must resolve to strings, numbers, or booleans.',
    );
  });

  it('validates and parses the binary-audio response parser contract', () => {
    expect(
      validateCustomTtsResponseParser(
        CUSTOM_TTS_RESPONSE_PARSER_BINARY_EXAMPLE,
      ),
    ).toBeUndefined();

    const parser = parseCustomTtsResponseParser(
      CUSTOM_TTS_RESPONSE_PARSER_BINARY_EXAMPLE,
    );

    expect(
      parseCustomTtsResponseFrame(new Uint8Array([1, 2, 3]), parser),
    ).toEqual({
      kind: 'audio',
      audio: new Uint8Array([1, 2, 3]),
      payload: new Uint8Array([1, 2, 3]),
    });

    expect(
      parseCustomTtsResponseFrame(
        '{"type":"done","message_id":"msg-7"}',
        parser,
      ),
    ).toEqual({
      kind: 'done',
      messageId: 'msg-7',
      payload: {
        type: 'done',
        message_id: 'msg-7',
      },
      done: true,
    });
  });

  it('parses JSON audio frames and error frames from ordered parser rules', () => {
    const parser = parseCustomTtsResponseParser(
      CUSTOM_TTS_RESPONSE_PARSER_JSON_AUDIO_EXAMPLE,
    );

    expect(
      parseCustomTtsResponseFrame(
        '{"type":"chunk","message_id":"msg-8","audio":"AAEC"}',
        parser,
      ),
    ).toEqual({
      kind: 'audio',
      messageId: 'msg-8',
      payload: {
        type: 'chunk',
        message_id: 'msg-8',
        audio: 'AAEC',
      },
      audio: new Uint8Array([0, 1, 2]),
    });

    expect(
      parseCustomTtsResponseFrame(
        '{"type":"error","message_id":"msg-9","error":{"message":"bad frame"}}',
        parser,
      ),
    ).toEqual({
      kind: 'error',
      messageId: 'msg-9',
      payload: {
        type: 'error',
        message_id: 'msg-9',
        error: {
          message: 'bad frame',
        },
      },
      error: 'bad frame',
      done: true,
    });
  });
});
