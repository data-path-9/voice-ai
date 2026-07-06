import React from 'react';
import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

export type NodeIconComponent = React.ComponentType<{
  size?: number;
  className?: string;
}>;

export type BaseNodeCardProps = {
  toolbar: React.ReactNode;
  handles: React.ReactNode;
  Icon: NodeIconComponent;
  cardClass: string;
  node: AgentflowNode;
  validationPill: React.ReactNode;
  width: number;
};
