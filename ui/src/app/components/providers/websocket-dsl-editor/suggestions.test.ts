import {
  extractWebsocketDslCastQuery,
  extractWebsocketDslVariableQuery,
  getWebsocketDslEditorSuggestions,
  shouldAutoTriggerWebsocketDslSuggestions,
} from './suggestions';

describe('websocket DSL editor suggestions', () => {
  it('detects DSL variable completion after $var', () => {
    expect(
      extractWebsocketDslVariableQuery('"message_id":{"$var":"'),
    ).toBe('');
    expect(
      extractWebsocketDslVariableQuery('"message_id":{"$var":"mess'),
    ).toBe('mess');
    expect(extractWebsocketDslVariableQuery('"message_id":"mess')).toBeNull();
  });

  it('detects cast completion after $cast', () => {
    expect(extractWebsocketDslCastQuery('"sample_rate":{"$cast":"')).toBe('');
    expect(extractWebsocketDslCastQuery('"sample_rate":{"$cast":"num')).toBe(
      'num',
    );
  });

  it('returns message_id variable suggestions for request mappings', () => {
    const suggestions = getWebsocketDslEditorSuggestions(
      'custom-tts',
      'text_request',
      '"message_id":{"$var":"mess',
    );

    expect(
      suggestions.some(
        item => item.label === 'message_id' && item.insertText === 'message_id',
      ),
    ).toBe(true);
  });

  it('returns response parser snippets at the start of the document', () => {
    const suggestions = getWebsocketDslEditorSuggestions(
      'custom-tts',
      'response_parser',
      '[',
    );

    expect(suggestions.some(item => item.label === 'Binary audio parser')).toBe(
      true,
    );
    expect(
      suggestions.some(item => item.label === 'JSON base64 audio parser'),
    ).toBe(true);
  });

  it('returns audio variable and transcript parser suggestions for custom-stt', () => {
    const variableSuggestions = getWebsocketDslEditorSuggestions(
      'custom-stt',
      'audio_request',
      '"audio":{"$var":"aud',
    );

    expect(
      variableSuggestions.some(
        item => item.label === 'audio' && item.insertText === 'audio',
      ),
    ).toBe(true);

    const parserSuggestions = getWebsocketDslEditorSuggestions(
      'custom-stt',
      'response_parser',
      '[',
    );

    expect(
      parserSuggestions.some(
        item => item.label === 'Plain text transcript parser',
      ),
    ).toBe(true);
  });

  it('auto-triggers request suggestions for snippets, $var, and $cast values', () => {
    expect(
      shouldAutoTriggerWebsocketDslSuggestions('text_request', '{'),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions(
        'text_request',
        '"text":{"$var":"',
      ),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions(
        'query_params',
        '"sample_rate":{"$cast":"',
      ),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions('response_parser', '['),
    ).toBe(true);
    expect(
      shouldAutoTriggerWebsocketDslSuggestions(
        'response_parser',
        '[{"when"',
      ),
    ).toBe(false);
  });
});
