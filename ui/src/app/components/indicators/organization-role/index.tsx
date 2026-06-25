import React from 'react';
import { Tag } from '@carbon/react';
import { UserAdmin, UserMultiple, UserRole } from '@carbon/icons-react';

export const OrganizationRoleIndicator = ({ role, size = 'medium' }) => {
  const roleConfig = {
    owner: {
      type: 'red',
      display: 'Owner',
      Icon: UserAdmin,
    },
    OWNER: {
      type: 'red',
      display: 'Owner',
      Icon: UserAdmin,
    },
    admin: {
      type: 'blue',
      display: 'Admin',
      Icon: UserRole,
    },
    ADMIN: {
      type: 'blue',
      display: 'Admin',
      Icon: UserRole,
    },
    member: {
      type: 'green',
      display: 'Member',
      Icon: UserMultiple,
    },
    MEMBER: {
      type: 'green',
      display: 'Member',
      Icon: UserMultiple,
    },
    DEFAULT: {
      type: 'gray',
      display: 'Member',
      Icon: UserMultiple,
    },
  };
  const config = role
    ? roleConfig[role] || roleConfig['DEFAULT']
    : roleConfig['DEFAULT'];
  const Icon = config.Icon;

  const sizeClasses = {
    small: 'sm',
    medium: 'md',
    large: 'md',
  };

  const sizeClass = sizeClasses[size] || sizeClasses.medium;

  return (
    <Tag
      size={sizeClass}
      type={config.type}
      renderIcon={Icon}
      className="!whitespace-nowrap"
    >
      {config.display}
    </Tag>
  );
};
