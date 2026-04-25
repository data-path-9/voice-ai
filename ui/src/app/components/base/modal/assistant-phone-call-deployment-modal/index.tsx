import {
  AssistantPhoneDeployment,
  DeploymentAudioProvider,
} from '@rapidaai/react';
import { ModalProps } from '@/app/components/base/modal';
import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { CONFIG } from '@/configs';
import { CopyButton } from '@/app/components/carbon/button/copy-button';
import { InputHelper } from '@/app/components/input-helper';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { ProviderPill } from '@/app/components/pill/provider-model-pill';
import { FC, ReactNode, useMemo, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  DeploymentRow,
  DeploymentSectionHeader,
} from '@/app/components/base/modal/deployment-modal-primitives';
import { Tabs } from '@/app/components/carbon/tabs';

interface AssistantPhoneCallDeploymentDialogProps extends ModalProps {
  deployment: AssistantPhoneDeployment;
}

export function AssistantPhoneCallDeploymentDialog(
  props: AssistantPhoneCallDeploymentDialogProps,
) {
  const [selectedTab, setSelectedTab] = useState(0);
  const providerName = props.deployment?.getPhoneprovidername()?.toLowerCase() || '';
  const assistantId = props.deployment?.getAssistantid();
  const mediaHost = CONFIG.connection.media;
  const sipHost = CONFIG.connection.sip;
  const socketHost = CONFIG.connection.socket;

  const webhookUrl = `${mediaHost}/v1/talk/${providerName || '<provider>'}/call/${assistantId}?x-api-key={{PROJECT_CRDENTIAL_KEY}}`;
  const eventUrl = `${mediaHost}/v1/talk/${providerName || '<provider>'}/event/${assistantId}?x-api-key={{PROJECT_CRDENTIAL_KEY}}`;

  const modalContent = (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-[580px]"
      label="Phone Deployment"
      title={props.deployment.getId()}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        <Tabs
          tabs={['Integration', 'Audio']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          aria-label="Phone deployment tabs"
          className="!h-full !min-h-0 !flex !flex-col [&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-gray-200 dark:[&_.cds--tabs__nav]:border-gray-800 [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0"
          panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
        >
          <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
            <TelephonyConfig deployment={props.deployment} />
            {providerName === 'sip' && (
              <SipIntegrationInstructions
                sipHost={sipHost}
                assistantId={assistantId}
              />
            )}
            {providerName === 'asterisk' && (
              <AsteriskIntegrationInstructions
                mediaHost={mediaHost}
                audioSocketHost={socketHost}
                assistantId={assistantId}
              />
            )}
            {providerName !== 'sip' && providerName !== 'asterisk' && (
              <>
                <CodeRow label="Inbound webhook url" value={webhookUrl}>
                  <InputHelper>
                    You can add additional agent arguments as query
                    parameters — e.g.{' '}
                    <code className="text-red-600">`?name=your-name`</code>
                  </InputHelper>
                </CodeRow>
                <CodeRow
                  label="Call status / Event callback webhook"
                  value={eventUrl}
                />
              </>
            )}
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-800 w-full">
            <VoiceInput deployment={props.deployment?.getInputaudio()} />
            <VoiceOutput deployment={props.deployment?.getOutputaudio()} />
          </div>
        </Tabs>
      </div>
    </RightSideModal>
  );

  if (typeof document === 'undefined') return modalContent;

  return createPortal(modalContent, document.body);
}

/* -------------------------------------------------------------------------- */
/*  Shared row primitives                                                      */
/* -------------------------------------------------------------------------- */

const Row = DeploymentRow;
const SectionHeader = DeploymentSectionHeader;

const TelephonyConfig: FC<{ deployment: AssistantPhoneDeployment }> = ({
  deployment,
}) => {
  const options = (deployment.getPhoneoptionsList() || []).filter(
    d => d.getKey() && d.getValue(),
  );

  return (
    <>
      <SectionHeader label="Telephony" />
      <Row label="Provider">
        <ProviderPill provider={deployment.getPhoneprovidername()} />
      </Row>
      {options.length > 0 ? (
        options.map((detail, index) => (
          <Row key={`phone-option-${index}`} label={detail.getKey()}>
            <span className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate max-w-[200px] text-right">
              {detail.getValue()}
            </span>
            <CopyButton className="h-6 w-6 shrink-0">
              {detail.getValue()}
            </CopyButton>
          </Row>
        ))
      ) : (
        <div className="px-4 py-3">
          <YellowNoticeBlock>No telephony options configured.</YellowNoticeBlock>
        </div>
      )}
    </>
  );
};

const CodeRow: FC<{ label: string; value: string; children?: ReactNode }> = ({
  label,
  value,
  children,
}) => (
  <div>
    <SectionHeader label={label} />
    <div className="px-4 py-3 space-y-2">
      <div className="flex items-center gap-2">
        <code className="flex-1 dark:bg-gray-950 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden break-all">
          {value}
        </code>
        <CopyButton className="h-7 w-7 shrink-0 border border-gray-200 dark:border-gray-800">
          {value}
        </CopyButton>
      </div>
      {children}
    </div>
  </div>
);

const CodeBlock: FC<{ label: string; code: string; helper?: ReactNode }> = ({
  label,
  code,
  helper,
}) => (
  <div>
    <SectionHeader label={label} />
    <div className="px-4 py-3 space-y-2">
      <div className="relative">
        <pre className="dark:bg-gray-950 bg-gray-100 px-3 py-2 font-mono text-xs overflow-auto">
          {code}
        </pre>
        <div className="absolute top-1 right-1">
          <CopyButton className="h-6 w-6 bg-gray-200 dark:bg-gray-800">
            {code}
          </CopyButton>
        </div>
      </div>
      {helper}
    </div>
  </div>
);

/* -------------------------------------------------------------------------- */
/*  SIP Provider Integration Instructions                                      */
/* -------------------------------------------------------------------------- */

const SipIntegrationInstructions: FC<{
  sipHost?: string;
  assistantId: string;
}> = ({ sipHost, assistantId }) => {
  const sipEndpoint = `sip:${sipHost}`;
  const sipUri = `sip:${assistantId || '{ASSISTANT_ID}'}:{{PROJECT_CREDENTIAL_KEY}}@${sipHost}`;

  return (
    <>
      <CodeRow label="SIP Server Endpoint" value={sipEndpoint}>
        <InputHelper>
          Point your SIP trunk / PBX outbound proxy to this address. Rapida
          accepts SIP INVITE and establishes an RTP media session directly.
        </InputHelper>
      </CodeRow>

      <CodeRow label="SIP URI (Authentication)" value={sipUri}>
        <InputHelper>
          Replace{' '}
          <code className="text-red-600">
            {'{{'}PROJECT_CREDENTIAL_KEY{'}}'}
          </code>{' '}
          with your project API key. Format:{' '}
          <code className="text-red-600">sip:assistantID:apiKey@host</code>
        </InputHelper>
      </CodeRow>

      <SectionHeader label="SIP Configuration" />
      <Row label="Transport">
        <span className="text-sm text-gray-900 dark:text-gray-100">
          UDP, TCP, or TLS
        </span>
      </Row>
      <Row label="Port">
        <span className="text-sm font-mono text-gray-900 dark:text-gray-100">
          5060
        </span>
      </Row>
      <Row label="Codec">
        <span className="text-sm text-gray-900 dark:text-gray-100 text-right">
          G.711 μ-law / A-law + DTMF
        </span>
      </Row>
      <Row label="Authentication">
        <span className="text-sm text-gray-900 dark:text-gray-100 text-right">
          URI-based (assistantID:apiKey)
        </span>
      </Row>
      <Row label="Media">
        <span className="text-sm text-gray-900 dark:text-gray-100">
          RTP (direct)
        </span>
      </Row>

      <CodeBlock
        label="PBX Dial Plan — FreeSWITCH"
        code={`<extension name="rapida-ai">
  <condition field="destination_number" expression="^(\\d+)$">
    <action application="bridge"
            data="sofia/external/${sipUri}"/>
  </condition>
</extension>`}
      />

      <CodeBlock
        label="PBX Dial Plan — Asterisk (pjsip.conf + extensions.conf)"
        code={`; pjsip.conf
[rapida-trunk]
type = endpoint
transport = transport-udp
context = from-rapida
aors = rapida-trunk
outbound_auth = rapida-trunk-auth

[rapida-trunk-auth]
type = auth
auth_type = userpass
username = ${assistantId || '{ASSISTANT_ID}'}
password = <YOUR_API_KEY>

[rapida-trunk]
type = aor
contact = sip:${sipHost}:5060

; extensions.conf
[rapida-outbound]
exten => _X.,1,Dial(PJSIP/\${EXTEN}@rapida-trunk)`}
      />
    </>
  );
};

/* -------------------------------------------------------------------------- */
/*  Asterisk Provider Integration Instructions                                 */
/* -------------------------------------------------------------------------- */

const AsteriskIntegrationInstructions: FC<{
  mediaHost: string;
  audioSocketHost?: string;
  assistantId: string;
}> = ({ mediaHost, audioSocketHost, assistantId }) => {
  const rapidaHostname = useMemo(() => {
    try {
      return new URL(mediaHost).hostname;
    } catch {
      return '<your-rapida-host>';
    }
  }, [mediaHost]);

  const audioSocketHostPart = useMemo(() => {
    if (!audioSocketHost) return '<your-rapida-host>';
    return audioSocketHost.split(':')[0] || '<your-rapida-host>';
  }, [audioSocketHost]);

  const audioSocketPort = useMemo(() => {
    if (!audioSocketHost) return '4573';
    const parts = audioSocketHost.split(':');
    return parts.length > 1 ? parts[1] : '4573';
  }, [audioSocketHost]);

  const webhookUrl = `https://${rapidaHostname}/v1/talk/asterisk/call/${assistantId || '{ASSISTANT_ID}'}?from=\${CALLERID(num)}&x-api-key={{PROJECT_CREDENTIAL_KEY}}`;

  return (
    <>
      <CodeRow
        label="WebSocket — Endpoint"
        value={`wss://${rapidaHostname}/v1/talk/asterisk/ctx/{contextId}`}
      />

      <CodeBlock
        label="WebSocket — Dialplan (extensions.conf)"
        code={`[rapida-inbound-ws]
exten => _X.,1,Answer()
 same => n,Set(CTX=\${CURL(${webhookUrl})})
 same => n,GotoIf($["\${CTX}" = ""]?error)
 same => n,WebSocket(wss://${rapidaHostname}/v1/talk/asterisk/ctx/\${CTX})
 same => n,Hangup()
 same => n(error),Playback(an-error-has-occurred)
 same => n,Hangup()`}
        helper={
          <InputHelper>
            Requires <code>chan_websocket.so</code> (Asterisk 20+). WSS port 443
            — ideal for cloud / NAT traversal.
          </InputHelper>
        }
      />

      <CodeRow label="AudioSocket — Endpoint" value={audioSocketHost ?? ''} />

      <CodeBlock
        label="AudioSocket — Dialplan (extensions.conf)"
        code={`[rapida-inbound]
exten => _X.,1,Answer()
 same => n,Set(CHANNEL(audioreadformat)=slin)
 same => n,Set(CHANNEL(audiowriteformat)=slin)
 same => n,Set(CTX=\${CURL(${webhookUrl})})
 same => n,GotoIf($["\${CTX}" = ""]?error)
 same => n,AudioSocket(\${CTX},${audioSocketHostPart}:${audioSocketPort})
 same => n,Hangup()
 same => n(error),Playback(an-error-has-occurred)
 same => n,Hangup()`}
        helper={
          <InputHelper>
            Requires <code>res_audiosocket.so</code> (Asterisk 16+). Raw TCP
            port {audioSocketPort} — SLIN 16-bit 8 kHz. Best for LAN / private
            network.
          </InputHelper>
        }
      />
    </>
  );
};

/* -------------------------------------------------------------------------- */
/*  Voice Input / Output helpers                                               */
/* -------------------------------------------------------------------------- */

const VoiceInput: FC<{ deployment?: DeploymentAudioProvider }> = ({
  deployment,
}) => (
  <>
    <SectionHeader label="Speech to text" />
    {deployment?.getAudiooptionsList() ? (
      deployment?.getAudiooptionsList().length > 0 && (
        <>
          <Row label="Provider">
            <ProviderPill provider={deployment?.getAudioprovider()} />
          </Row>
          {deployment
            ?.getAudiooptionsList()
            .filter(d => d.getValue())
            .filter(d => d.getKey().startsWith('listen.'))
            .map((detail, index) => (
              <Row key={index} label={detail.getKey()}>
                <span className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate max-w-[200px] text-right">
                  {detail.getValue()}
                </span>
                <CopyButton className="h-6 w-6 shrink-0">
                  {detail.getValue()}
                </CopyButton>
              </Row>
            ))}
        </>
      )
    ) : (
      <div className="px-4 py-3">
        <YellowNoticeBlock>Voice input is not enabled</YellowNoticeBlock>
      </div>
    )}
  </>
);

const VoiceOutput: FC<{ deployment?: DeploymentAudioProvider }> = ({
  deployment,
}) => (
  <>
    <SectionHeader label="Text to speech" />
    {deployment?.getAudiooptionsList() ? (
      deployment?.getAudiooptionsList().length > 0 && (
        <>
          <Row label="Provider">
            <ProviderPill provider={deployment?.getAudioprovider()} />
          </Row>
          {deployment
            ?.getAudiooptionsList()
            .filter(d => d.getValue())
            .filter(d => d.getKey().startsWith('speak.'))
            .map((detail, index) => (
              <Row key={index} label={detail.getKey()}>
                <span className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate max-w-[200px] text-right">
                  {detail.getValue()}
                </span>
                <CopyButton className="h-6 w-6 shrink-0">
                  {detail.getValue()}
                </CopyButton>
              </Row>
            ))}
        </>
      )
    ) : (
      <div className="px-4 py-3">
        <YellowNoticeBlock>Voice output is not enabled</YellowNoticeBlock>
      </div>
    )}
  </>
);
