import { useEffect, useState, useCallback } from 'react';
import { Helmet } from '@/app/components/helmet';
import { InviteOrganizationUserDialog } from '@/app/components/base/modal/invite-organization-user-modal';
import { InviteProjectUserDialog } from '@/app/components/base/modal/invite-project-user-modal';
import {
  DeleteUserFromOrganization,
  DeleteUserFromOrganizationRequest,
  User,
} from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { useUserPageStore } from '@/hooks';
import { SingleUser } from '@/app/pages/workspace/user/single-user';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Pagination } from '@/app/components/carbon/pagination';
import { Add, Renew, TrashCan } from '@carbon/icons-react';
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableToolbar,
  TableBatchAction,
  TableBatchActions,
  TableToolbarContent,
  TableToolbarSearch,
  Button,
} from '@carbon/react';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import { TableSection } from '@/app/components/sections/table-section';
import { ConfirmDeleteDialog } from '@/app/components/base/modal/confirm-delete';
import { connectionConfig } from '@/configs';

const headers = [
  { key: 'id', header: 'ID' },
  { key: 'name', header: 'Name' },
  { key: 'email', header: 'Email' },
  { key: 'role', header: 'Org Role' },
  { key: 'createdDate', header: 'Date Created' },
  { key: 'status', header: 'Status' },
];

export function UserPage() {
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [inviteOrganizationModalOpen, setInviteOrganizationModalOpen] =
    useState(false);
  const [inviteProjectModalOpen, setInviteProjectModalOpen] = useState(false);
  const [userPendingDelete, setUserPendingDelete] = useState<User | null>(null);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const { projectId, authId, token } = useCurrentCredential();
  const userActions = useUserPageStore();
  const selectedUser = userActions.users.find(
    user => user.getId() === selectedUserId,
  );

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: User[]) => {
    hideLoader();
  }, []);

  const getUsers = useCallback((token, userId, projectId) => {
    showLoader();
    userActions.getAllUser(token, userId, projectId, onError, onSuccess);
  }, []);

  useEffect(() => {
    getUsers(token, authId, projectId);
  }, [userActions.page, userActions.pageSize, userActions.criteria]);

  const onDeleteOrganizationUser = async (user: User) => {
    showLoader('overlay');
    const req = new DeleteUserFromOrganizationRequest();
    req.setUserid(user.getId());

    try {
      const response = await DeleteUserFromOrganization(connectionConfig, req, {
        authorization: token,
        'x-auth-id': authId,
      });
      hideLoader();

      const responseError = response.getError();
      const message =
        responseError?.getHumanmessage() ||
        'Unable to process your request. please try again later.';

      if (response.getSuccess()) {
        setUserPendingDelete(null);
        setSelectedUserId(null);
        toast.success('The user was deleted from the organization.');
        getUsers(token, authId, projectId);
        return;
      }

      toast.error(message);
    } catch (err: any) {
      hideLoader();
      toast.error(
        err?.message ||
          'Unable to process your request. please try again later.',
      );
    }
  };

  return (
    <>
      <Helmet title="User and Teams" />
      <PageHeaderBlock>
        <PageTitleWithCount
          count={userActions.users.length}
          total={userActions.totalCount}
        >
          Users
        </PageTitleWithCount>
      </PageHeaderBlock>
      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={Boolean(selectedUser)}
          totalSelected={selectedUser ? 1 : 0}
          onCancel={() => setSelectedUserId(null)}
          totalCount={userActions.users.length}
        >
          <TableBatchAction
            renderIcon={Add}
            onClick={() => {
              if (selectedUser) {
                setInviteProjectModalOpen(true);
              }
            }}
          >
            Invite to project
          </TableBatchAction>
          <TableBatchAction
            className="cds--btn--danger"
            renderIcon={TrashCan}
            onClick={() => {
              if (selectedUser) {
                setUserPendingDelete(selectedUser);
              }
            }}
          >
            Delete
          </TableBatchAction>
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search users..." />
          <Button
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            kind="ghost"
            onClick={() => getUsers(token, authId, projectId)}
            tooltipPosition="bottom"
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            isLoading={loading}
            onClick={() => setInviteOrganizationModalOpen(true)}
          >
            Invite user
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>
      <TableSection>
        <Table>
          <TableHead>
            <TableRow>
              <TableHeader className="!w-12" />
              {headers.map(h => (
                <TableHeader key={h.key}>{h.header}</TableHeader>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {userActions.users.map((usr, idx) => (
              <SingleUser
                key={idx}
                user={usr}
                selected={selectedUserId === usr.getId()}
                onSelect={() =>
                  setSelectedUserId(
                    selectedUserId === usr.getId() ? null : usr.getId(),
                  )
                }
              />
            ))}
          </TableBody>
        </Table>
        <Pagination
          totalItems={userActions.totalCount}
          page={userActions.page}
          pageSize={userActions.pageSize}
          pageSizes={[10, 20, 50]}
          onChange={({ page, pageSize }) => {
            if (pageSize !== userActions.pageSize) {
              userActions.setPageSize(pageSize);
            } else {
              userActions.setPage(page);
            }
          }}
        />
      </TableSection>
      <InviteOrganizationUserDialog
        modalOpen={inviteOrganizationModalOpen}
        setModalOpen={setInviteOrganizationModalOpen}
        onSuccess={() => {
          getUsers(token, authId, projectId);
        }}
      />
      <InviteProjectUserDialog
        modalOpen={inviteProjectModalOpen}
        setModalOpen={open => {
          setInviteProjectModalOpen(open);
          if (!open) {
            setSelectedUserId(null);
          }
        }}
        user={selectedUser || null}
        onSuccess={() => {
          setSelectedUserId(null);
          getUsers(token, authId, projectId);
        }}
      />
      <ConfirmDeleteDialog
        showing={Boolean(userPendingDelete)}
        title="Delete user"
        content={
          userPendingDelete
            ? `This will delete "${userPendingDelete.getEmail()}" from the organization. Type the email to confirm.`
            : ''
        }
        objectName={userPendingDelete?.getEmail() || ''}
        confirmText="Delete user"
        onConfirm={() => {
          if (userPendingDelete) {
            onDeleteOrganizationUser(userPendingDelete);
          }
        }}
        onCancel={() => {
          setUserPendingDelete(null);
          setSelectedUserId(null);
        }}
        onClose={() => {
          setUserPendingDelete(null);
          setSelectedUserId(null);
        }}
      />
    </>
  );
}
