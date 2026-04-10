import React, { HTMLAttributes } from 'react';
import { cn } from '@/utils';
import { useSidebar } from '@/context/sidebar-context';
import { Text } from '@/app/components/carbon/text';

interface SidebarLabelProps extends HTMLAttributes<HTMLSpanElement> {
  isLoading?: boolean;
}

export function SidebarLabel({ isLoading, ...props }: SidebarLabelProps) {
  const { open } = useSidebar();

  return (
    <span
      className={cn(
        'text-sm truncate flex-1 transition-all duration-200 font-semibold',
        open ? 'opacity-100' : 'opacity-0 w-0',
        props.className,
      )}
    >
      <Text isLoading={isLoading} skeletonWidth="70%">
        {props.children}
      </Text>
    </span>
  );
}
