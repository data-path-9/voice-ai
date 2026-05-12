import { useParams } from 'react-router-dom';

import {
  CreateAuthenticationForm,
  EditAuthenticationForm,
} from './authentication-form';
import { ConfigureAuthenticationList } from './configure-authentication-list';

export function ConfigureAssistantAuthenticationPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAuthenticationList assistantId={assistantId} />}
    </>
  );
}

export function CreateAssistantAuthenticationPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <CreateAuthenticationForm assistantId={assistantId} />}</>
  );
}

export function UpdateAssistantAuthenticationPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <EditAuthenticationForm assistantId={assistantId} />}</>
  );
}
