import { Metadata } from '@rapidaai/react';
import { loadProviderConfig } from '../config-loader';
import { getDefaultsFromConfig, validateFromConfig } from '../config-defaults';
import {
  CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE,
  CUSTOM_TTS_QUERY_PARAMS_KEY,
  CUSTOM_TTS_REQUEST_RULES_KEY,
  CUSTOM_TTS_RESPONSE_RULES_KEY,
} from '../custom-tts/contract';

function createMetadata(key: string, value: string): Metadata {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
}

function findMeta(source: Metadata[], key: string): string | undefined {
  return source.find(item => item.getKey() === key)?.getValue();
}

function upsertMetadata(
  source: Metadata[],
  key: string,
  value: string,
): Metadata[] {
  const next = source.filter(item => item.getKey() !== key);
  next.push(createMetadata(key, value));
  return next;
}

function removeMetadata(source: Metadata[], key: string): Metadata[] {
  return source.filter(item => item.getKey() !== key);
}

function getProvider(list: any[]) {
  return list.find(provider => provider.code === 'custom-tts');
}

describe('Custom TTS provider catalog', () => {
  it('exists in development and production with websocket credential fields', () => {
    const developmentProviders = require('../provider.development.json');
    const productionProviders = require('../provider.production.json');

    const developmentProvider = getProvider(developmentProviders);
    const productionProvider = getProvider(productionProviders);

    for (const provider of [developmentProvider, productionProvider]) {
      expect(provider).toBeDefined();
      expect(provider.featureList).toEqual(
        expect.arrayContaining(['tts', 'external']),
      );
      expect(provider.configurations.map((config: any) => config.name)).toEqual(
        expect.arrayContaining(['apiCompatibility', 'baseUrl', 'headers']),
      );
      const compatibilityConfig = provider.configurations.find(
        (config: any) => config.name === 'apiCompatibility',
      );
      expect(
        compatibilityConfig?.choices?.map((choice: any) => choice.value),
      ).toEqual(['websocket_v1']);
    }
  });
});

describe('Custom TTS config contract', () => {
  const config = loadProviderConfig('custom-tts')!;
  const validQueryParams =
    '{"language":{"$var":"language"},"message_id":{"$var":"message_id"},"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}';
  const validTwoStepRequestRules =
    '[{"when":{"packet":"text"},"send":{"frame":"json","body":{"text":{"$path":"packet.text"},"voice_id":{"$path":"config.voice.id"},"message_id":{"$path":"packet.message_id"}}}},{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done","message_id":{"$path":"packet.message_id"}}}}]';
  const validResponseRules =
    '[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}},{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"message_id"},"done":true}}]';
  const validJsonResponseRules =
    '[{"when":{"frame":"json","path":"type","equals":"chunk"},"emit":{"audio":{"$decode":"base64","value":{"$path":"audio"}},"message_id":{"$path":"message_id"}}},{"when":{"frame":"json","path":"type","equals":"done"},"emit":{"message_id":{"$path":"message_id"},"done":true}}]';

  const buildValidOptions = (): Metadata[] => {
    const defaults = getDefaultsFromConfig(
      config,
      'tts',
      [createMetadata('rapida.credential_id', 'cred-custom-1')],
      'custom-tts',
    );

    return upsertMetadata(
      upsertMetadata(defaults, CUSTOM_TTS_QUERY_PARAMS_KEY, validQueryParams),
      CUSTOM_TTS_RESPONSE_RULES_KEY,
      validResponseRules,
    );
  };

  it('loads the expected canonical TTS metadata keys', () => {
    expect(config.tts).toBeDefined();
    const keys = config.tts?.parameters.map(param => param.key) ?? [];
    expect(keys).toEqual(
      expect.arrayContaining([
        'speak.audio.encoding',
        'speak.audio.sample_rate',
        CUSTOM_TTS_QUERY_PARAMS_KEY,
        CUSTOM_TTS_REQUEST_RULES_KEY,
        CUSTOM_TTS_RESPONSE_RULES_KEY,
      ]),
    );
    expect(keys).not.toContain('speak.ws.text_request');
    expect(keys).not.toContain('speak.ws.done_request');
    expect(keys).not.toContain('speak.ws.response_parser');
    expect(keys).not.toContain('speak.model');
    expect(keys).not.toContain('speak.language');
    expect(keys).not.toContain('speak.voice.id');
  });

  it('applies encoding, sample-rate, and request-rule defaults', () => {
    const defaults = getDefaultsFromConfig(
      config,
      'tts',
      [createMetadata('rapida.credential_id', 'cred-custom-1')],
      'custom-tts',
    );

    expect(findMeta(defaults, 'speak.audio.encoding')).toBe('LINEAR16');
    expect(findMeta(defaults, 'speak.audio.sample_rate')).toBe('16000');
    expect(findMeta(defaults, CUSTOM_TTS_REQUEST_RULES_KEY)).toBe(
      CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE,
    );
  });

  it.each([
    [
      'speak.audio.encoding',
      'Please select a valid audio encoding for custom TTS.',
    ],
    [
      'speak.audio.sample_rate',
      'Please select a valid sample rate for custom TTS.',
    ],
    [
      CUSTOM_TTS_REQUEST_RULES_KEY,
      'Please provide valid request rules for custom TTS.',
    ],
    [
      CUSTOM_TTS_RESPONSE_RULES_KEY,
      'Please provide valid response rules for custom TTS.',
    ],
  ])('requires %s', (key, expectedError) => {
    const result = validateFromConfig(
      config,
      'tts',
      'custom-tts',
      removeMetadata(buildValidOptions(), key),
    );

    expect(result).toBe(expectedError);
  });

  it('accepts a valid canonical websocket TTS DSL configuration', () => {
    expect(
      validateFromConfig(config, 'tts', 'custom-tts', buildValidOptions()),
    ).toBeUndefined();
  });

  it('also accepts two-step request rules and JSON audio response rules', () => {
    const options = upsertMetadata(
      upsertMetadata(
        buildValidOptions(),
        CUSTOM_TTS_REQUEST_RULES_KEY,
        validTwoStepRequestRules,
      ),
      CUSTOM_TTS_RESPONSE_RULES_KEY,
      validJsonResponseRules,
    );

    expect(
      validateFromConfig(config, 'tts', 'custom-tts', options),
    ).toBeUndefined();
  });

  it('allows query params to be omitted', () => {
    const options = buildValidOptions().filter(
      item => item.getKey() !== CUSTOM_TTS_QUERY_PARAMS_KEY,
    );

    expect(
      validateFromConfig(config, 'tts', 'custom-tts', options),
    ).toBeUndefined();
  });

  it('rejects invalid JSON in optional query params', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_TTS_QUERY_PARAMS_KEY),
      CUSTOM_TTS_QUERY_PARAMS_KEY,
      '{"language":{"$var":"language"',
    );

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBe(
      'Please provide a valid JSON definition for query parameters.',
    );
  });

  it('rejects query params that use removed text variable', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_TTS_QUERY_PARAMS_KEY),
      CUSTOM_TTS_QUERY_PARAMS_KEY,
      '{"text":{"$var":"text"}}',
    );

    const result = validateFromConfig(config, 'tts', 'custom-tts', options);
    expect(result).toContain('Unsupported custom tts variable "text"');
  });

  it('rejects invalid request rules JSON', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_TTS_REQUEST_RULES_KEY),
      CUSTOM_TTS_REQUEST_RULES_KEY,
      '[{"when":{"packet":"text"}}',
    );

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBe(
      'Please provide valid JSON request rules for custom TTS.',
    );
  });

  it('rejects request rules without a text packet', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_TTS_REQUEST_RULES_KEY),
      CUSTOM_TTS_REQUEST_RULES_KEY,
      '[{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done"}}}]',
    );

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBe(
      'Custom TTS request rules must contain at least one rule with when.packet "text".',
    );
  });

  it('rejects invalid response rules JSON', () => {
    const options = upsertMetadata(
      removeMetadata(buildValidOptions(), CUSTOM_TTS_RESPONSE_RULES_KEY),
      CUSTOM_TTS_RESPONSE_RULES_KEY,
      '[{"when":',
    );

    expect(validateFromConfig(config, 'tts', 'custom-tts', options)).toBe(
      'Please provide a valid JSON response rules for custom TTS.',
    );
  });
});
