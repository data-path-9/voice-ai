import type { FC, ReactNode } from 'react';
import {
  Breadcrumb as CarbonBreadcrumb,
  BreadcrumbItem as CarbonBreadcrumbItem,
  BreadcrumbSkeleton,
} from '@carbon/react';
import { cn } from '@/utils';

// ─── Types ───────────────────────────────────────────────────────────────────

export interface BreadcrumbItemData {
  label: ReactNode;
  href?: string;
  /** Render custom content inside the BreadcrumbItem instead of a plain link. */
  render?: () => ReactNode;
}

export interface CarbonBreadcrumbProps {
  items: BreadcrumbItemData[];
  className?: string;
  noTrailingSlash?: boolean;
  isLoading?: boolean;
}

// ─── Breadcrumb ──────────────────────────────────────────────────────────────

/** Carbon Breadcrumb — renders items or a skeleton placeholder when loading. */
export const Breadcrumb: FC<CarbonBreadcrumbProps> = ({
  items,
  className,
  noTrailingSlash = true,
  isLoading = false,
}) => {
  if (isLoading) {
    return <BreadcrumbSkeleton className={cn(className)} />;
  }

  return (
    <CarbonBreadcrumb noTrailingSlash={noTrailingSlash} className={cn(className)}>
      {items.map((item, idx) => (
        <CarbonBreadcrumbItem key={idx} href={item.render ? undefined : item.href}>
          {item.render ? item.render() : item.label}
        </CarbonBreadcrumbItem>
      ))}
    </CarbonBreadcrumb>
  );
};

export { CarbonBreadcrumbItem as BreadcrumbItem };
