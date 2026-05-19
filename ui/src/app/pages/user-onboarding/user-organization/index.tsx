import React, { useCallback, useContext, useState } from 'react';
import { Helmet } from '@/app/components/helmet';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { CreateOrganization } from '@rapidaai/react';
import { CreateOrganizationResponse } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import { connectionConfig } from '@/configs';
import { Stack, TextInput } from '@/app/components/carbon/form';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Notification } from '@/app/components/carbon/notification';
import { ArrowRight } from '@carbon/icons-react';
import { Select, SelectItem } from '@carbon/react';

export function CreateOrganizationPage() {
  const navigate = useNavigate();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { authorize } = useContext(AuthContext);
  const { user, authId, token } = useCurrentCredential();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm();
  const [error, setError] = useState('');

  const afterCreateOrganization = useCallback(
    (err: ServiceError | null, org: CreateOrganizationResponse | null) => {
      if (err) {
        hideLoader();
        setError('Unable to process your request. Please try again later.');
        return;
      }
      if (org?.getSuccess()) {
        authorize &&
          authorize(
            () => { hideLoader(); return navigate('/onboarding/project'); },
            () => { hideLoader(); setError('Please provide valid credentials to sign in.'); },
          );
      } else {
        hideLoader();
        setError('Please provide valid credentials to sign in.');
      }
    },
    [],
  );

  const onCreateOrganization = data => {
    showLoader('overlay');
    CreateOrganization(
      connectionConfig,
      data.organizationName,
      data.organizationSize,
      data.organizationIndustry,
      { authorization: token, 'x-auth-id': authId },
      afterCreateOrganization,
    );
  };

  const formError =
    (errors.organizationName?.message as string) ||
    (errors.organizationSize?.message as string) ||
    (errors.organizationIndustry?.message as string) ||
    error;

  return (
    <>
      <Helmet title="Onboarding: Create an organization" />
      <div className="mb-4">
        <h1 className="text-xl font-light tracking-tight">Set up your organization</h1>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
          Create the top-level workspace that owns assistants, credentials,
          client programs, and governance settings.
        </p>
      </div>

      <form onSubmit={handleSubmit(onCreateOrganization)}>
        <Stack gap={5}>
          <TextInput
            id="org-name"
            labelText="Organization Name"
            type="text"
            required
            defaultValue={`${user?.name}'s Organization`}
            placeholder="eg: Acme Voice Studio"
            helperText="Use your agency, brand, or operating company name."
            {...register('organizationName', { required: 'Please enter the organization name.' })}
          />
          <Select
            id="org-size"
            labelText="Operating Model"
            helperText="Helps tailor onboarding for agency delivery, internal platform teams, and enterprise programs."
            {...register('organizationSize')}
          >
            <SelectItem value="" text="Select your operating model" />
            <SelectItem value="agency" text="Agency / system integrator" />
            <SelectItem value="in-house" text="In-house product team" />
            <SelectItem value="enterprise" text="Enterprise / multi-team org" />
          </Select>
          <TextInput
            id="org-industry"
            labelText="Industry"
            type="text"
            required
            placeholder="eg: Agency services, healthcare, finance"
            helperText="Used to suggest integrations and assistant templates for your market."
            {...register('organizationIndustry', { required: 'Please provide an industry.' })}
          />
          {formError && (
            <Notification kind="error" title="Error" subtitle={formError} />
          )}
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            type="submit"
            isLoading={loading}
            className="!w-full !max-w-none !justify-between"
          >
            Continue
          </PrimaryButton>
        </Stack>
      </form>
    </>
  );
}
