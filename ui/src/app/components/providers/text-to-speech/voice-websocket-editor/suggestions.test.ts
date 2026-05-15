import {
  extractVoiceWebsocketCastQuery,
  extractVoiceWebsocketVariableQuery,
  getVoiceWebsocketEditorSuggestions,
  shouldAutoTriggerVoiceWebsocketSuggestions,
} from './suggestions';

describe('voice websocket editor suggestions', () => {
  it('detects DSL variable completion after $var', () => {
    expect(
      extractVoiceWebsocketVariableQuery('"message_id":{"$var":"'),
    ).toBe('');
    expect(
      extractVoiceWebsocketVariableQuery('"message_id":{"$var":"mess'),
    ).toBe('mess');
    expect(extractVoiceWebsocketVariableQuery('"message_id":"mess')).toBeNull();
  });

  it('detects cast completion after $cast', () => {
    expect(extractVoiceWebsocketCastQuery('"sample_rate":{"$cast":"')).toBe('');
    expect(extractVoiceWebsocketCastQuery('"sample_rate":{"$cast":"num')).toBe(
      'num',
    );
  });

  it('returns message_id variable suggestions for request mappings', () => {
    const suggestions = getVoiceWebsocketEditorSuggestions(
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
    const suggestions = getVoiceWebsocketEditorSuggestions(
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

  it('auto-triggers request suggestions for snippets, $var, and $cast values', () => {
    expect(
      shouldAutoTriggerVoiceWebsocketSuggestions('text_request', '{'),
    ).toBe(true);
    expect(
      shouldAutoTriggerVoiceWebsocketSuggestions(
        'text_request',
        '"text":{"$var":"',
      ),
    ).toBe(true);
    expect(
      shouldAutoTriggerVoiceWebsocketSuggestions(
        'query_params',
        '"sample_rate":{"$cast":"',
      ),
    ).toBe(true);
    expect(
      shouldAutoTriggerVoiceWebsocketSuggestions('response_parser', '['),
    ).toBe(true);
    expect(
      shouldAutoTriggerVoiceWebsocketSuggestions(
        'response_parser',
        '[{"when"',
      ),
    ).toBe(false);
  });
});
