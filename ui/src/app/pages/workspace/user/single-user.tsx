import React from 'react';
import { TextImage } from '@/app/components/text-image';
import { User } from '@rapidaai/react';
import { OrganizationRoleIndicator } from '@/app/components/indicators/organization-role';
import { toHumanReadableDate } from '@/utils/date';
import { RadioButton, TableRow, TableCell } from '@carbon/react';
import { CarbonIconIndicator } from '@/app/components/carbon/icon-indicator';
import { CopyButton } from '@/app/components/carbon/button/copy-button';

export function SingleUser(props: {
  user: User;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <TableRow
      isSelected={props.selected}
      onClick={props.onSelect}
      className="cursor-pointer"
    >
      <TableCell className="!w-12 !pr-0" onClick={e => e.stopPropagation()}>
        <RadioButton
          id={`user-select-${props.user.getId()}`}
          name="user-select"
          labelText=""
          hideLabel
          checked={props.selected}
          onChange={props.onSelect}
        />
      </TableCell>
      <TableCell>{props.user.getId()}</TableCell>
      <TableCell>
        <div className="flex items-center gap-3">
          <TextImage size={7} name={props.user.getName()} />
          <span>{props.user.getName()}</span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2 min-w-0 max-w-[280px]">
          <span className="truncate">{props.user.getEmail()}</span>
          <div className="shrink-0" onClick={e => e.stopPropagation()}>
            <CopyButton className="h-6 w-6">{props.user.getEmail()}</CopyButton>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <OrganizationRoleIndicator role={props.user.getRole()} />
      </TableCell>
      <TableCell>
        {props.user.getCreateddate() &&
          toHumanReadableDate(props.user.getCreateddate()!)}
      </TableCell>
      <TableCell>
        <CarbonIconIndicator state={props.user.getStatus?.()} />
      </TableCell>
    </TableRow>
  );
}
