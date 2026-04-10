import type { FC } from 'react';
import { ToastNotification as CarbonToastNotification } from '@carbon/react';
import { cn } from '@/utils';

// ─── Types ───────────────────────────────────────────────────────────────────

type ToastKind =
  | 'info'
  | 'success'
  | 'warning'
  | 'error'
  | 'info-square'
  | 'warning-alt';

// ─── Toast Notification ─────────────────────────────────────────────────────

export interface ToastNotificationProps {
  kind?: ToastKind;
  title: string;
  subtitle?: string;
  caption?: string;
  className?: string;
  lowContrast?: boolean;
  hideCloseButton?: boolean;
  timeout?: number;
  role?: 'alert' | 'log' | 'status';
  statusIconDescription?: string;
  onClose?: () => void;
  onCloseButtonClick?: () => void;
}

/** Carbon ToastNotification — floating notification toast. */
export const ToastNotification: FC<ToastNotificationProps> = ({
  kind = 'info',
  title,
  subtitle,
  caption,
  className,
  lowContrast = true,
  hideCloseButton = false,
  timeout = 0,
  role = 'status',
  statusIconDescription = 'notification',
  onClose,
  onCloseButtonClick,
}) => (
  <CarbonToastNotification
    kind={kind}
    title={title}
    subtitle={subtitle}
    caption={caption}
    lowContrast={lowContrast}
    hideCloseButton={hideCloseButton}
    timeout={timeout}
    role={role}
    statusIconDescription={statusIconDescription}
    onClose={onClose}
    onCloseButtonClick={onCloseButtonClick}
    className={cn('!max-w-full', className)}
  />
);
