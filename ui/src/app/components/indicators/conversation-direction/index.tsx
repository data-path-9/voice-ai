import React from 'react';
import { Tag } from '@carbon/react';
import { ArrowDown, ArrowUp } from '@carbon/icons-react';

interface ConversationDirectionIndicatorProps {
  direction: string;
}

export const ConversationDirectionIndicator: React.FC<
  ConversationDirectionIndicatorProps
> = ({ direction }) => {
  const isInbound = direction?.toLowerCase() !== 'outbound';

  return (
    <Tag
      size="md"
      type={isInbound ? 'green' : 'warm-gray'}
      className="!inline-flex !items-center"
    >
      <span className="inline-flex items-center gap-1.5 leading-none [&>svg]:block">
        {isInbound ? <ArrowDown size={16} /> : <ArrowUp size={16} />}
        {isInbound ? 'Inbound' : 'Outbound'}
      </span>
    </Tag>
  );
};
