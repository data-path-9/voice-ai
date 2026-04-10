import { Argument } from '@rapidaai/react';
import { Table } from '@/app/components/base/tables/table';
import { TableBody } from '@/app/components/base/tables/table-body';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { TableHead } from '@/app/components/base/tables/table-head';
import { TableRow } from '@/app/components/base/tables/table-row';
import { FC } from 'react';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { DataCheck } from '@carbon/icons-react';

export const EndpointArguments: FC<{ args: Array<Argument> }> = ({ args }) => {
  if (args.length <= 0)
    return (
      <EmptyState
        className="h-full min-h-[420px]"
        icon={DataCheck}
        title="No arguments found"
        subtitle="No runtime arguments were recorded for this trace."
      />
    );
  return (
    <Table className="w-full">
      <TableHead
        columns={[
          { name: 'Name', key: 'Name' },
          { name: 'Value', key: 'Value' },
        ]}
      />
      <TableBody>
        {args.map((ar, index) => {
          return (
            <TableRow key={index}>
              <TableCell>{ar.getName()}</TableCell>
              <TableCell className="break-words break-all">
                {ar.getValue()}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
};
