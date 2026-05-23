import type { FC, ReactNode } from 'react';
import { Link, TableCell } from '@carbon/react';

import { CopyButton } from '@/app/components/carbon/button/copy-button';
import { cn } from '@/utils';

interface UrlTableCellProps {
  url: string;
  copyText?: string;
  prefix?: ReactNode;
  cellClassName?: string;
  maxWidthClassName?: string;
  valueClassName?: string;
  copyButtonClassName?: string;
}

export const UrlTableCell: FC<UrlTableCellProps> = ({
  url,
  copyText,
  prefix,
  cellClassName,
  maxWidthClassName = 'max-w-[360px]',
  valueClassName,
  copyButtonClassName = 'h-6 w-6 shrink-0',
}) => {
  const valueToCopy = copyText || url;

  return (
    <TableCell className={cn('text-sm', cellClassName)}>
      <div className={cn('flex items-center gap-2 min-w-0', maxWidthClassName)}>
        {prefix}
        <Link
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className={cn('truncate min-w-0 !block', valueClassName)}
          onClick={e => e.stopPropagation()}
        >
          {url}
        </Link>
        <div className="shrink-0" onClick={e => e.stopPropagation()}>
          <CopyButton className={copyButtonClassName}>{valueToCopy}</CopyButton>
        </div>
      </div>
    </TableCell>
  );
};
