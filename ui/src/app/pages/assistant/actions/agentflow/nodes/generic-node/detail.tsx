import React from 'react';

export type GenericNodeDetailProps = {
  children: React.ReactNode;
};

export function GenericNodeDetail({ children }: GenericNodeDetailProps) {
  return <>{children}</>;
}
