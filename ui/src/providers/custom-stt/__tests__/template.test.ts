import {
  CUSTOM_STT_AUDIO_REQUEST_EXAMPLE,
  CUSTOM_STT_DSL_VARIABLES,
  CUSTOM_STT_QUERY_PARAMS_EXAMPLE,
  CUSTOM_STT_RESPONSE_PARSER_EXAMPLE,
  CUSTOM_STT_RESPONSE_PARSER_JSON_EXAMPLE,
  CUSTOM_STT_RESPONSE_PARSER_NESTED_EXAMPLE,
  parseCustomSttResponseFrame,
  parseCustomSttResponseParser,
  renderCustomSttQueryParams,
  renderCustomSttRequestDefinition,
  validateCustomSttQueryParams,
  validateCustomSttRequestDefinition,
  validateCustomSttResponseParser,
} from '../contract';

describe('custom-stt websocket DSL helpers', () => {
  it('renders request bindings and query params from the shared DSL', () => {
    const renderedRequest = renderCustomSttRequestDefinition(
      CUSTOM_STT_AUDIO_REQUEST_EXAMPLE,
      {
        audio: 'AAEC',
        model: 'nova-3',
        language: 'en-US',
        encoding: 'LINEAR16',
        sample_rate: '16000',
      },
    );

    expect(renderedRequest).toEqual({
      audio: 'AAEC',
      encoding: 'LINEAR16',
      sample_rate: 16000,
    });

    expect(
      renderCustomSttQueryParams(CUSTOM_STT_QUERY_PARAMS_EXAMPLE, {
        audio: 'AAEC',
        model: 'nova-3',
        language: 'en-US',
        encoding: 'LINEAR16',
        sample_rate: '16000',
      }),
    ).toEqual({
      language: 'en-US',
      model: 'nova-3',
      encoding: 'LINEAR16',
      sample_rate: 16000,
    });
  });

  it('rejects unsupported DSL variables and query param objects that do not resolve to primitives', () => {
    const variableResult = validateCustomSttRequestDefinition(
      '{"audio":{"$var":"chunk"}}',
      'Audio Request',
    );

    expect(variableResult).toContain('"chunk"');
    for (const variable of CUSTOM_STT_DSL_VARIABLES) {
      expect(variableResult).toContain(variable);
    }

    expect(
      validateCustomSttQueryParams(
        '{"audio":{"payload":{"$var":"audio"}}}',
        'Query Parameters',
      ),
    ).toBe(
      'Custom STT query parameters values must resolve to strings, numbers, or booleans.',
    );
  });

  it('validates and parses plain text transcript parser rules', () => {
    expect(
      validateCustomSttResponseParser(CUSTOM_STT_RESPONSE_PARSER_EXAMPLE),
    ).toBeUndefined();

    const parser = parseCustomSttResponseParser(
      CUSTOM_STT_RESPONSE_PARSER_EXAMPLE,
    );

    expect(
      parseCustomSttResponseFrame(
        'namaste',
        parser,
      ),
    ).toEqual({
      kind: 'transcript',
      payload: 'namaste',
      script: 'namaste',
      language: 'hi',
      interim: true,
    });
  });

  it('validates and parses interim/final JSON transcript parser rules', () => {
    expect(
      validateCustomSttResponseParser(CUSTOM_STT_RESPONSE_PARSER_JSON_EXAMPLE),
    ).toBeUndefined();

    const parser = parseCustomSttResponseParser(
      CUSTOM_STT_RESPONSE_PARSER_JSON_EXAMPLE,
    );

    expect(
      parseCustomSttResponseFrame(
        '{"type":"partial","text":"hello","confidence":"0.72","language":"en-US"}',
        parser,
      ),
    ).toEqual({
      kind: 'transcript',
      payload: {
        type: 'partial',
        text: 'hello',
        confidence: '0.72',
        language: 'en-US',
      },
      script: 'hello',
      confidence: 0.72,
      language: 'en-US',
      interim: true,
    });

    expect(
      parseCustomSttResponseFrame(
        '{"type":"error","error":{"message":"bad frame"}}',
        parser,
      ),
    ).toEqual({
      kind: 'error',
      payload: {
        type: 'error',
        error: {
          message: 'bad frame',
        },
      },
      error: 'bad frame',
    });
  });

  it('parses nested transcript payloads from ordered parser rules', () => {
    const parser = parseCustomSttResponseParser(
      CUSTOM_STT_RESPONSE_PARSER_NESTED_EXAMPLE,
    );

    expect(
      parseCustomSttResponseFrame(
        '{"result":{"final":true,"transcript":"bonjour","confidence":"0.93","language":"fr-FR"}}',
        parser,
      ),
    ).toEqual({
      kind: 'transcript',
      payload: {
        result: {
          final: true,
          transcript: 'bonjour',
          confidence: '0.93',
          language: 'fr-FR',
        },
      },
      script: 'bonjour',
      confidence: 0.93,
      language: 'fr-FR',
      interim: false,
    });
  });

  it('rejects text parser rules that try to use when.path', () => {
    expect(
      validateCustomSttResponseParser(
        '[{"when":{"frame":"text","path":"type","equals":"partial"},"emit":{"script":{"$frame":"text"},"interim":true}}]',
      ),
    ).toBe(
      'Custom STT response parser rule 1 cannot use when.path with text frames.',
    );
  });
});
