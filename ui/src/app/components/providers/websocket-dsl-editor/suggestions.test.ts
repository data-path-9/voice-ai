import {
  extractWebsocketDslCastQuery,
  extractWebsocketDslPathQuery,
  extractWebsocketDslVariableQuery,
  getWebsocketDslEditorSuggestions,
  shouldAutoTriggerWebsocketDslSuggestions,
} from './suggestions';

describe('websocket DSL editor suggestions', () => {
  it('detects DSL variable completion after $var', () => {
    expect(extractWebsocketDslVariableQuery('"message_id":{"$var":"')).toBe('');
    expect(extractWebsocketDslVariableQuery('"message_id":{"$var":"mess')).toBe(
      'mess',
    );
    expect(extractWebsocketDslVariableQuery('"message_id":"mess')).toBeNull();
  });

  it('detects path completion after $path', () => {
    expect(extractWebsocketDslPathQuery('"body":{"$path":"')).toBe('');
    expect(
      extractWebsocketDslPathQuery('"body":{"$path":"packet.audio.ba'),
    ).toBe('packet.audio.ba');
  });

  it('detects cast completion after $cast', () => {
    expect(extractWebsocketDslCastQuery('"sample_rate":{"$cast":"')).toBe('');
    expect(extractWebsocketDslCastQuery('"sample_rate":{"$cast":"num')).toBe(
      'num',
    );
  });

  it('returns message_id variable suggestions for TTS query params', () => {
    const suggestions = getWebsocketDslEditorSuggestions(
      'custom-tts',
      'query_params',
      '"message_id":{"$var":"mess',
    );

    expect(
      suggestions.some(
        item => item.label === 'message_id' && item.insertText === 'message_id',
      ),
    ).toBe(true);
  });

  it('does not suggest removed text variable in TTS query params', () => {
    const suggestions = getWebsocketDslEditorSuggestions(
      'custom-tts',
      'query_params',
      '"message":{"$var":"t',
    );

    expect(suggestions.some(item => item.label === 'text')).toBe(false);
  });

  it('returns request-rule path suggestions for custom-stt', () => {
    const suggestions = getWebsocketDslEditorSuggestions(
      'custom-stt',
      'request_rules',
      '"body":{"$path":"packet.audio.ba',
    );

    expect(
      suggestions.some(
        item =>
          item.label === 'packet.audio.base64' &&
          item.insertText === 'packet.audio.base64',
      ),
    ).toBe(true);
  });

  it('returns request-rule path suggestions for custom-tts', () => {
    const suggestions = getWebsocketDslEditorSuggestions(
      'custom-tts',
      'request_rules',
      '"body":{"$path":"config.voice.',
    );

    expect(
      suggestions.some(
        item =>
          item.label === 'config.voice.id' &&
          item.insertText === 'config.voice.id',
      ),
    ).toBe(true);
  });

  it('returns response-rule snippets at the start of the document', () => {
    const suggestions = getWebsocketDslEditorSuggestions(
      'custom-tts',
      'response_rules',
      '[',
    );

    expect(suggestions.some(item => item.label === 'Binary audio parser')).toBe(
      true,
    );
    expect(
      suggestions.some(item => item.label === 'JSON base64 audio parser'),
    ).toBe(true);
  });

  it('returns request-rule and response-rule snippets for custom-stt', () => {
    const requestSuggestions = getWebsocketDslEditorSuggestions(
      'custom-stt',
      'request_rules',
      '[',
    );

    expect(
      requestSuggestions.some(item => item.label === 'Binary audio rule'),
    ).toBe(true);
    expect(
      requestSuggestions.some(item => item.label === 'Turn-change start rule'),
    ).toBe(true);

    const responseSuggestions = getWebsocketDslEditorSuggestions(
      'custom-stt',
      'response_rules',
      '[',
    );

    expect(
      responseSuggestions.some(
        item => item.label === 'Plain text transcript parser',
      ),
    ).toBe(true);
  });

  it('auto-triggers request and response suggestions for the right DSL shapes', () => {
    expect(shouldAutoTriggerWebsocketDslSuggestions('query_params', '{')).toBe(
      true,
    );
    expect(
      shouldAutoTriggerWebsocketDslSuggestions(
        'query_params',
        '"text":{"$var":"',
      ),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions(
        'request_rules',
        '"body":{"$path":"',
      ),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions(
        'request_rules',
        '"sample_rate":{"$cast":"',
      ),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions('response_rules', '['),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions('response_rules', '[{"when"'),
    ).toBe(false);
  });
});
