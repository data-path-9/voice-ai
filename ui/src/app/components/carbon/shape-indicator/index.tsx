import { FC } from 'react';
import { preview__ShapeIndicator as ShapeIndicator } from '@carbon/react';

export type CarbonShapeIndicatorKind =
  | 'failed'
  | 'critical'
  | 'high'
  | 'medium'
  | 'low'
  | 'cautious'
  | 'undefined'
  | 'stable'
  | 'informative'
  | 'incomplete'
  | 'draft';

export interface CarbonShapeIndicatorProps {
  state?: string;
  kind?: CarbonShapeIndicatorKind;
  label?: string;
  textSize?: 12 | 14;
}

export const CarbonShapeIndicator: FC<CarbonShapeIndicatorProps> = ({
  state,
  kind,
  label,
  textSize = 12,
}) => {
  const resolved = state
    ? recordStateToShapeIndicator[state] || defaultRecordShapeIndicator
    : {
        kind: kind || defaultRecordShapeIndicator.kind,
        label: label || defaultRecordShapeIndicator.label,
      };

  return (
    <ShapeIndicator
      kind={resolved.kind as any}
      label={resolved.label}
      textSize={textSize}
    />
  );
};

export const recordStateToShapeIndicator: Record<
  string,
  { kind: CarbonShapeIndicatorKind; label: string }
> = {
  RECORD_ACTIVE: { kind: 'stable', label: 'Active' },
  ACTIVE: { kind: 'stable', label: 'Active' },
  active: { kind: 'stable', label: 'Active' },

  RECORD_INVITED: { kind: 'cautious', label: 'Invited' },
  INVITED: { kind: 'cautious', label: 'Invited' },
  invited: { kind: 'cautious', label: 'Invited' },

  RECORD_QUEUED: { kind: 'medium', label: 'Queued' },
  QUEUED: { kind: 'medium', label: 'Queued' },
  queued: { kind: 'medium', label: 'Queued' },

  RECORD_CONNECTED: { kind: 'stable', label: 'Connected' },
  CONNECTED: { kind: 'stable', label: 'Connected' },
  connected: { kind: 'stable', label: 'Connected' },

  RECORD_IN_PROGRESS: { kind: 'informative', label: 'In Progress' },
  IN_PROGRESS: { kind: 'informative', label: 'In Progress' },
  in_progress: { kind: 'informative', label: 'In Progress' },

  RECORD_SUCCESS: { kind: 'stable', label: 'Success' },
  SUCCESS: { kind: 'stable', label: 'Success' },
  success: { kind: 'stable', label: 'Success' },

  RECORD_COMPLETE: { kind: 'stable', label: 'Complete' },
  COMPLETE: { kind: 'stable', label: 'Complete' },
  complete: { kind: 'stable', label: 'Complete' },
  COMPLETED: { kind: 'stable', label: 'Completed' },
  completed: { kind: 'stable', label: 'Completed' },

  RECORD_INACTIVE: { kind: 'draft', label: 'Inactive' },
  INACTIVE: { kind: 'draft', label: 'Inactive' },
  inactive: { kind: 'draft', label: 'Inactive' },

  RECORD_ARCHIEVE: { kind: 'draft', label: 'Archived' },
  ARCHIEVE: { kind: 'draft', label: 'Archived' },
  archived: { kind: 'draft', label: 'Archived' },

  RECORD_FAILED: { kind: 'failed', label: 'Failed' },
  FAILED: { kind: 'failed', label: 'Failed' },
  failed: { kind: 'failed', label: 'Failed' },
};

export const defaultRecordShapeIndicator = {
  kind: 'undefined' as CarbonShapeIndicatorKind,
  label: 'Unknown',
};
