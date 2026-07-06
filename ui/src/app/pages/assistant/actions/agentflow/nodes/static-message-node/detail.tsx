import React from 'react';

export type StaticMessageNodeDetailProps = {
  children: React.ReactNode;
};

export function StaticMessageNodeDetail({
  children,
}: StaticMessageNodeDetailProps) {
  return <>{children}</>;
}
