import {
  CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE,
  CUSTOM_STT_DSL_VARIABLES,
  CUSTOM_STT_QUERY_PARAMS_EXAMPLE,
  CUSTOM_STT_REQUEST_RULES_JSON_AUDIO_EXAMPLE,
  CUSTOM_STT_RESPONSE_RULES_EXAMPLE,
  CUSTOM_STT_RESPONSE_RULES_JSON_EXAMPLE,
  CUSTOM_STT_RESPONSE_RULES_NESTED_EXAMPLE,
  parseCustomSttRequestRules,
  parseCustomSttResponseFrame,
  parseCustomSttResponseRules,
  renderCustomSttQueryParams,
  renderCustomSttRequestRuleBody,
  validateCustomSttQueryParams,
  validateCustomSttRequestRules,
  validateCustomSttResponseRules,
} from '../contract';

describe('custom-stt websocket DSL helpers', () => {
  it('renders query params from the shared DSL', () => {
    expect(
      renderCustomSttQueryParams(CUSTOM_STT_QUERY_PARAMS_EXAMPLE, {
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

  it('rejects unsupported query-param variables and nested object values', () => {
    const variableResult = validateCustomSttQueryParams(
      '{"audio":{"$var":"audio"}}',
      'Query Parameters',
    );
    expect(variableResult).toContain('"audio"');
    for (const variable of CUSTOM_STT_DSL_VARIABLES) {
      expect(variableResult).toContain(variable);
    }

    expect(
      validateCustomSttQueryParams(
        '{"model":{"payload":{"$var":"model"}}}',
        'Query Parameters',
      ),
    ).toBe(
      'Custom STT query parameters values must resolve to strings, numbers, or booleans.',
    );
  });

  it('validates and renders binary request rules', () => {
    expect(
      validateCustomSttRequestRules(CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE),
    ).toBeUndefined();

    const rules = parseCustomSttRequestRules(
      CUSTOM_STT_DEFAULT_REQUEST_RULES_EXAMPLE,
    );

    expect(rules).toHaveLength(1);
    expect(rules[0].send.frame).toBe('binary');
    expect(
      renderCustomSttRequestRuleBody(rules[0], {
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
          },
        },
      }),
    ).toBe('AAEC');
  });

  it('validates and renders JSON request rules', () => {
    expect(
      validateCustomSttRequestRules(
        CUSTOM_STT_REQUEST_RULES_JSON_AUDIO_EXAMPLE,
      ),
    ).toBeUndefined();

    const rules = parseCustomSttRequestRules(
      CUSTOM_STT_REQUEST_RULES_JSON_AUDIO_EXAMPLE,
    );

    expect(
      renderCustomSttRequestRuleBody(rules[0], {
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
          },
        },
      }),
    ).toEqual({
      audio: 'AAEC',
      encoding: 'LINEAR16',
      sample_rate: 16000,
    });
  });

  it('validates and renders WAV base64 request paths', () => {
    const rules = parseCustomSttRequestRules(
      '[{"when":{"packet":"audio"},"send":{"frame":"json","body":{"audio":{"$path":"packet.audio.wav_base64"}}}}]',
    );

    expect(
      renderCustomSttRequestRuleBody(rules[0], {
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
      }),
    ).toEqual({ audio: 'UklGRg==' });
  });

  it('rejects request rules that use unsupported $path roots', () => {
    expect(
      validateCustomSttRequestRules(
        '[{"when":{"packet":"audio"},"send":{"frame":"binary","body":{"$path":"payload.audio"}}}]',
      ),
    ).toBe(
      'Custom STT request rules only supports "$path" roots of config, packet.',
    );
  });

  it('rejects request rules without an audio packet rule', () => {
    expect(
      validateCustomSttRequestRules(
        '[{"when":{"packet":"turn_change"},"send":{"frame":"json","body":{"type":"start"}}}]',
      ),
    ).toBe(
      'Custom STT request rules must contain at least one rule with when.packet "audio".',
    );
  });

  it('validates and parses plain text transcript response rules', () => {
    expect(
      validateCustomSttResponseRules(CUSTOM_STT_RESPONSE_RULES_EXAMPLE),
    ).toBeUndefined();

    const parser = parseCustomSttResponseRules(
      CUSTOM_STT_RESPONSE_RULES_EXAMPLE,
    );

    expect(parseCustomSttResponseFrame('namaste', parser)).toEqual({
      kind: 'transcript',
      payload: 'namaste',
      script: 'namaste',
      language: 'hi',
      interim: true,
    });
  });

  it('validates and parses interim/final JSON transcript response rules', () => {
    expect(
      validateCustomSttResponseRules(CUSTOM_STT_RESPONSE_RULES_JSON_EXAMPLE),
    ).toBeUndefined();

    const parser = parseCustomSttResponseRules(
      CUSTOM_STT_RESPONSE_RULES_JSON_EXAMPLE,
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

  it('parses nested transcript payloads from ordered response rules', () => {
    const parser = parseCustomSttResponseRules(
      CUSTOM_STT_RESPONSE_RULES_NESTED_EXAMPLE,
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

  it('rejects text response rules that try to use when.path', () => {
    expect(
      validateCustomSttResponseRules(
        '[{"when":{"frame":"text","path":"type","equals":"partial"},"emit":{"script":{"$frame":"text"},"interim":true}}]',
      ),
    ).toBe(
      'Custom STT response rules rule 1 cannot use when.path with text frames.',
    );
  });
});
