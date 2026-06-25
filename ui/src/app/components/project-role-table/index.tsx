import React from 'react';
import { Add, TrashCan } from '@carbon/icons-react';
import { Button, Dropdown } from '@carbon/react';
import { TertiaryButton } from '@/app/components/carbon/button';

export type ProjectRoleRow = {
  projectId: string;
  projectRole: string;
};

type ProjectOption = {
  name: string;
  value: string;
};

type RoleOption = {
  name: string;
  value: string;
};

interface ProjectRoleTableProps {
  rows: ProjectRoleRow[];
  onChange: (rows: ProjectRoleRow[]) => void;
  projectOptions: ProjectOption[];
  roleOptions: RoleOption[];
  title?: string;
  addButtonLabel?: string;
  showAddButton?: boolean;
  showRemoveColumn?: boolean;
  defaultProjectId?: string;
}

export function ProjectRoleTable(props: ProjectRoleTableProps) {
  const showAddButton = props.showAddButton ?? true;
  const showRemoveColumn = props.showRemoveColumn ?? true;
  const addButtonLabel = props.addButtonLabel || 'Add project role';

  return (
    <div>
      {props.title && (
        <p className="text-xs font-medium mb-2">
          {props.title} ({props.rows.length})
        </p>
      )}
      <table className="w-full border-collapse border border-gray-200 dark:border-gray-700 text-sm [&_.cds--list-box]:!border-none [&_.cds--form-item]:!m-0">
        <thead>
          <tr className="bg-gray-50 dark:bg-gray-900">
            <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/2">
              Project
            </th>
            <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/2">
              Role
            </th>
            {showRemoveColumn && (
              <th className="border-b border-gray-200 dark:border-gray-700 w-8" />
            )}
          </tr>
        </thead>
        <tbody>
          {props.rows.length === 0 && (
            <tr>
              <td
                colSpan={showRemoveColumn ? 3 : 2}
                className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400"
              >
                No project roles yet. Click <strong>{addButtonLabel}</strong>{' '}
                below.
              </td>
            </tr>
          )}
          {props.rows.map((row, index) => (
            <tr
              key={index}
              className="border-b border-gray-200 dark:border-gray-700 last:border-b-0"
            >
              <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                <Dropdown
                  id={`project-role-project-${index}`}
                  titleText=""
                  label="Select project"
                  items={props.projectOptions}
                  selectedItem={
                    props.projectOptions.find(
                      project => project.value === row.projectId,
                    ) || null
                  }
                  itemToString={(item: ProjectOption | null) =>
                    item?.name || ''
                  }
                  onChange={({ selectedItem }) => {
                    const next = [...props.rows];
                    next[index] = {
                      ...row,
                      projectId: selectedItem?.value || '',
                    };
                    props.onChange(next);
                  }}
                  size="md"
                  direction="top"
                  className="!w-full"
                />
              </td>
              <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                <Dropdown
                  id={`project-role-role-${index}`}
                  titleText=""
                  label="Select role"
                  items={props.roleOptions}
                  selectedItem={
                    props.roleOptions.find(
                      role => role.value === row.projectRole,
                    ) || null
                  }
                  itemToString={(item: RoleOption | null) => item?.name || ''}
                  onChange={({ selectedItem }) => {
                    const next = [...props.rows];
                    next[index] = {
                      ...row,
                      projectRole: selectedItem?.value || '',
                    };
                    props.onChange(next);
                  }}
                  size="md"
                  direction="top"
                  className="!w-full"
                />
              </td>
              {showRemoveColumn && (
                <td className="p-0 text-center">
                  <Button
                    hasIconOnly
                    renderIcon={TrashCan}
                    iconDescription="Remove project role"
                    kind="danger--ghost"
                    size="sm"
                    onClick={() =>
                      props.onChange(props.rows.filter((_, i) => i !== index))
                    }
                  />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      {showAddButton && (
        <div className="pt-2">
          <TertiaryButton
            size="md"
            renderIcon={Add}
            onClick={() =>
              props.onChange([
                ...props.rows,
                {
                  projectId:
                    props.defaultProjectId ||
                    props.projectOptions[0]?.value ||
                    '',
                  projectRole: '',
                },
              ])
            }
            className="!w-full !max-w-none"
          >
            {addButtonLabel}
          </TertiaryButton>
        </div>
      )}
    </div>
  );
}
