import React, { useContext, useEffect, useState } from 'react';
import {
  AddUserToProjects,
  AddUserToProjectsRequest,
  ProjectRoleAssignment,
  User,
} from '@rapidaai/react';
import { ComboBox } from '@carbon/react';
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
import { useRapidaStore, useUserPageStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import {
  ProjectRoleRow,
  ProjectRoleTable,
} from '@/app/components/project-role-table';

const projectRoles = [
  { name: 'Super Admin', value: 'super admin' },
  { name: 'Admin', value: 'admin' },
  { name: 'Writer', value: 'writer' },
  { name: 'Reader', value: 'reader' },
];

interface InviteProjectUserDialogProps extends ModalProps {
  user?: User | null;
  projectId?: string;
  onSuccess?: () => void;
}

export function InviteProjectUserDialog(props: InviteProjectUserDialogProps) {
  const { authId, token, projectId: currentProjectId } = useCurrentCredential();
  const { projectRoles: availableProjects } = useContext(AuthContext);
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const userActions = useUserPageStore();
  const [selectedUser, setSelectedUser] = useState<User | null>(
    props.user || null,
  );
  const [projectRoleRows, setProjectRoleRows] = useState<ProjectRoleRow[]>(
    props.projectId ? [{ projectId: props.projectId, projectRole: '' }] : [],
  );
  const [error, setError] = useState('');

  useEffect(() => {
    if (props.modalOpen) {
      setSelectedUser(props.user || null);
      setProjectRoleRows(
        props.projectId
          ? [{ projectId: props.projectId, projectRole: '' }]
          : [],
      );
      setError('');

      if (!props.user && userActions.users.length === 0) {
        userActions.getAllUser(
          token,
          authId,
          currentProjectId,
          err => setError(err),
          () => {},
        );
      }
    }
  }, [props.modalOpen, props.user, props.projectId]);

  const submitInvite = async () => {
    if (!selectedUser) {
      setError('Please select a user.');
      return;
    }
    if (projectRoleRows.length === 0) {
      setError('Please add at least one project role.');
      return;
    }
    if (projectRoleRows.some(row => !row.projectId || !row.projectRole)) {
      setError('Please select project and role.');
      return;
    }
    const projectIds = projectRoleRows.map(row => row.projectId);
    if (new Set(projectIds).size !== projectIds.length) {
      setError('A project can only be assigned once.');
      return;
    }

    setError('');
    showLoader('overlay');

    const req = new AddUserToProjectsRequest();
    req.setUserid(selectedUser.getId());
    req.setProjectrolesList(
      projectRoleRows.map(row => {
        const assignment = new ProjectRoleAssignment();
        assignment.setProjectid(row.projectId);
        assignment.setProjectrole(row.projectRole);
        return assignment;
      }),
    );

    try {
      const response = await AddUserToProjects(connectionConfig, req, {
        authorization: token,
        'x-auth-id': authId,
      });
      hideLoader();

      const responseError = response.getError();
      const message =
        responseError?.getHumanmessage() ||
        'Unable to process your request. please try again later.';

      if (response.getSuccess()) {
        setSelectedUser(null);
        setProjectRoleRows(
          props.projectId
            ? [
                {
                  projectId: props.projectId,
                  projectRole: '',
                },
              ]
            : [],
        );
        props.setModalOpen(false);
        toast.success('The user was added to the project successfully.');
        if (props.onSuccess) props.onSuccess();
        return;
      }

      toast.error(message);
      setError(message);
    } catch (err: any) {
      hideLoader();
      const message =
        err?.message ||
        'Unable to process your request. please try again later.';
      toast.error(message);
      setError(message);
    }
  };

  const projectOptions = props.projectId
    ? [
        {
          name:
            (availableProjects || []).find(
              project => project.projectid === props.projectId,
            )?.projectname || props.projectId,
          value: props.projectId,
        },
      ]
    : (availableProjects || []).map(project => ({
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
        label="Project Access"
        title="Invite user to project"
        onClose={() => props.setModalOpen(false)}
      />
      <ModalBody hasForm hasScrollingContent>
        <Stack gap={6}>
          {props.user ? (
            <TextInput
              id="project-invite-selected-user"
              labelText="User"
              value={`${props.user.getName()} (${props.user.getEmail()})`}
              readOnly
            />
          ) : (
            <ComboBox
              id="project-invite-user"
              titleText="User"
              placeholder="Select user"
              items={userActions.users}
              selectedItem={selectedUser}
              itemToString={(item: User | null) =>
                item ? `${item.getName()} (${item.getEmail()})` : ''
              }
              onChange={({ selectedItem }) => {
                setError('');
                setSelectedUser(selectedItem || null);
              }}
            />
          )}
          <ProjectRoleTable
            rows={projectRoleRows}
            onChange={next => {
              setError('');
              setProjectRoleRows(next);
            }}
            projectOptions={projectOptions}
            roleOptions={projectRoles}
            defaultProjectId={props.projectId || projectOptions[0]?.value || ''}
            title="Project roles"
            addButtonLabel="Add project role"
            showAddButton={!props.projectId}
            showRemoveColumn={!props.projectId}
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
