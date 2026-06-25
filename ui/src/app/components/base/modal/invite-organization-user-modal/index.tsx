import React, { useContext, useState } from 'react';
import {
  InviteUserToOrganization,
  InviteUserToOrganizationRequest,
  ProjectRoleAssignment,
} from '@rapidaai/react';
import { Dropdown } from '@carbon/react';
import toast from 'react-hot-toast/headless';
import { ModalProps } from '@/app/components/base/modal';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/carbon/modal';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { Stack, TextInput } from '@/app/components/carbon/form';
import { ErrorMessage } from '@/app/components/form/error-message';
import { AuthContext } from '@/context/auth-context';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import {
  ProjectRoleRow,
  ProjectRoleTable,
} from '@/app/components/project-role-table';

const organizationRoles = [
  { name: 'Owner', value: 'owner' },
  { name: 'Admin', value: 'admin' },
  { name: 'Member', value: 'member' },
];
const projectRoles = [
  { name: 'Super Admin', value: 'super admin' },
  { name: 'Admin', value: 'admin' },
  { name: 'Writer', value: 'writer' },
  { name: 'Reader', value: 'reader' },
];

interface InviteOrganizationUserDialogProps extends ModalProps {
  onSuccess?: () => void;
}

export function InviteOrganizationUserDialog(
  props: InviteOrganizationUserDialogProps,
) {
  const { authId, token } = useCurrentCredential();
  const { projectRoles: availableProjects } = useContext(AuthContext);
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [email, setEmail] = useState('');
  const [organizationRole, setOrganizationRole] = useState('');
  const [projectRoleRows, setProjectRoleRows] = useState<ProjectRoleRow[]>([]);
  const [error, setError] = useState('');

  const resetForm = () => {
    setEmail('');
    setOrganizationRole('');
    setProjectRoleRows([]);
    setError('');
  };

  const submitInvite = async () => {
    if (!email) {
      setError('Please provide a valid email to invite user.');
      return;
    }
    if (!organizationRole) {
      setError('Please select an organization role.');
      return;
    }
    if (projectRoleRows.some(row => !row.projectId || !row.projectRole)) {
      setError('Select both project and role for each project role row.');
      return;
    }
    const selectedProjectIds = projectRoleRows
      .filter(row => row.projectId)
      .map(row => row.projectId);
    if (new Set(selectedProjectIds).size !== selectedProjectIds.length) {
      setError('A project can only be assigned once.');
      return;
    }

    setError('');
    showLoader('overlay');

    const req = new InviteUserToOrganizationRequest();
    req.setEmail(email);
    req.setOrganizationrole(organizationRole);
    req.setProjectrolesList(
      projectRoleRows
        .filter(row => row.projectId && row.projectRole)
        .map(row => {
          const assignment = new ProjectRoleAssignment();
          assignment.setProjectid(row.projectId);
          assignment.setProjectrole(row.projectRole);
          return assignment;
        }),
    );

    try {
      const response = await InviteUserToOrganization(connectionConfig, req, {
        authorization: token,
        'x-auth-id': authId,
      });
      hideLoader();

      const responseError = response.getError();
      const message =
        responseError?.getHumanmessage() ||
        'Unable to process your request. please try again later.';

      if (response.getSuccess()) {
        resetForm();
        props.setModalOpen(false);
        toast.success('The organization invitation was sent successfully.');
        if (props.onSuccess) props.onSuccess();
        return;
      }

      toast.error(message);
      setError(message);
    } catch (err: any) {
      hideLoader();
      const message =
        err?.getHumanmessage?.() ||
        err?.message ||
        'Unable to process your request. please try again later.';
      toast.error(message);
      setError(message);
    }
  };

  const projectOptions = (availableProjects || []).map(project => ({
    name: project.projectname,
    value: project.projectid,
  }));

  return (
    <Modal
      open={props.modalOpen}
      onClose={() => props.setModalOpen(false)}
      size="md"
      preventCloseOnClickOutside
    >
      <ModalHeader
        label="User Management"
        title="Invite user to organization"
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody hasForm hasScrollingContent>
        <Stack gap={6}>
          <TextInput
            id="organization-invite-email"
            labelText="Email address"
            value={email}
            type="email"
            placeholder="eg: john@deo.io"
            onChange={e => {
              setError('');
              setEmail(e.target.value);
            }}
          />
          <Dropdown
            id="organization-invite-role"
            titleText="Organization role"
            label="Select organization role"
            items={organizationRoles}
            selectedItem={
              organizationRoles.find(role => role.value === organizationRole) ||
              null
            }
            itemToString={(item: { name: string; value: string } | null) =>
              item?.name || ''
            }
            onChange={({ selectedItem }) => {
              setError('');
              setOrganizationRole(selectedItem?.value || '');
            }}
          />
          <ProjectRoleTable
            rows={projectRoleRows}
            onChange={next => {
              setError('');
              setProjectRoleRows(next);
            }}
            projectOptions={projectOptions}
            roleOptions={projectRoles}
            defaultProjectId={projectOptions[0]?.value || ''}
            title="Project roles"
            addButtonLabel="Add project role"
          />
          <ErrorMessage message={error} />
        </Stack>
      </ModalBody>
      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => props.setModalOpen(false)}>
          Cancel
        </SecondaryButton>
        <PrimaryButton size="lg" onClick={submitInvite} isLoading={loading}>
          Invite user
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
