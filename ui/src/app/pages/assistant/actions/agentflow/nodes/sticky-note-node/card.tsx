import { useState } from 'react';
import MarkdownPreview from '@uiw/react-markdown-preview';
import { NodeResizer } from 'reactflow';
import type { AgentflowNode } from '@/app/pages/assistant/actions/agentflow';

type StickyNoteNodeCardProps = {
  node: AgentflowNode;
  selected: boolean;
  note: string;
  noteColor: string;
  onChangeNote: (note: string) => void;
  onResizeEnd: (size: { width: number; height: number }) => void;
};

const stickyNoteColors: Record<
  string,
  { background: string; fold: string; edge: string }
> = {
  yellow: {
    background: '#ffe982',
    fold: '#f3d150',
    edge: '#efd461',
  },
  blue: {
    background: '#d6eaff',
    fold: '#9fc9f5',
    edge: '#b7d8f8',
  },
  green: {
    background: '#d9f7c9',
    fold: '#a9de8f',
    edge: '#beeaa8',
  },
  purple: {
    background: '#eadcff',
    fold: '#c8adf3',
    edge: '#d9c5fa',
  },
};

export function StickyNoteNodeCard({
  node,
  selected,
  note,
  noteColor,
  onChangeNote,
  onResizeEnd,
}: StickyNoteNodeCardProps) {
  const [editing, setEditing] = useState(false);
  const noteValue = note || '# Quick start';
  const colors = stickyNoteColors[noteColor] ?? stickyNoteColors.yellow;

  return (
    <>
      <NodeResizer
        isVisible={selected}
        minWidth={280}
        minHeight={128}
        color="#0f62fe"
        lineClassName="!border !border-gray-600"
        lineStyle={{ borderWidth: 1, borderColor: '#6f6f6f' }}
        handleClassName="!rounded-none"
        handleStyle={{
          width: 8,
          height: 8,
          borderRadius: 0,
          border: '1px solid #ffffff',
          backgroundColor: '#0f62fe',
        }}
        onResizeEnd={(_, size) =>
          onResizeEnd({ width: size.width, height: size.height })
        }
      />
      <div
        data-testid="note_node"
        className="relative h-full w-full overflow-hidden rounded-xl text-left shadow-[0_18px_28px_rgba(0,0,0,0.18)] transition-shadow hover:shadow-[0_22px_34px_rgba(0,0,0,0.2)]"
        style={{
          backgroundColor: colors.background,
          borderTop: `1px solid ${colors.edge}`,
        }}
        onDoubleClick={() => setEditing(true)}
      >
        <div
          className="pointer-events-none absolute right-0 top-0 h-14 w-14"
          style={{
            background: `linear-gradient(135deg, ${colors.fold} 0 50%, rgba(255,255,255,0.36) 50% 100%)`,
            boxShadow: '-2px 2px 5px rgba(0,0,0,0.12)',
          }}
        />
        <div
          className="pointer-events-none absolute inset-x-0 top-0 h-3"
          style={{
            background:
              'linear-gradient(180deg, rgba(255,255,255,0.42), rgba(255,255,255,0))',
          }}
        />
        <div className="flex h-full min-h-0 p-2 pr-14">
          {editing ? (
            <textarea
              id={`${node.id}-sticky-note`}
              value={noteValue}
              placeholder="# Quick start"
              onChange={event => onChangeNote(event.target.value)}
              onBlur={() => setEditing(false)}
              autoFocus
              rows={4}
              aria-label="Sticky note"
              className="nodrag nopan block h-full min-h-0 w-full flex-1 resize-none self-stretch border-0 bg-transparent p-0 text-3xl font-bold leading-10 !text-black outline-none placeholder:text-black/60 focus:outline-none"
            />
          ) : (
            <div className="h-full w-full overflow-auto">
              <MarkdownPreview
                source={noteValue}
                className="!bg-transparent !text-black [&_*]:!text-black [&_.wmde-markdown]:!bg-transparent [&_.wmde-markdown]:!text-black [&_a]:!underline [&_code]:!bg-black/10 [&_h1]:!mb-0 [&_h1]:!text-5xl [&_h1]:!font-bold [&_h1]:!leading-tight [&_h2]:!text-4xl [&_p]:!text-2xl [&_p]:!leading-9"
                style={{ background: 'transparent', color: '#000000' }}
              />
            </div>
          )}
        </div>
      </div>
    </>
  );
}
