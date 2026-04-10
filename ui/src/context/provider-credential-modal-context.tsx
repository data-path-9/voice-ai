import React, { createContext, useContext, useMemo, useState } from 'react';
import { CreateProviderCredentialDialog } from '@/app/components/base/modal/create-provider-credential-modal';

type ProviderCredentialModalContextValue = {
  openCreateCredentialModal: (provider?: string | null) => void;
};

const ProviderCredentialModalContext =
  createContext<ProviderCredentialModalContextValue | null>(null);

export const ProviderCredentialModalProvider: React.FC<{
  children: React.ReactNode;
}> = ({ children }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [provider, setProvider] = useState<string | null>(null);

  const value = useMemo<ProviderCredentialModalContextValue>(
    () => ({
      openCreateCredentialModal: (nextProvider?: string | null) => {
        setProvider(nextProvider ?? null);
        setModalOpen(true);
      },
    }),
    [],
  );

  return (
    <ProviderCredentialModalContext.Provider value={value}>
      {children}
      <CreateProviderCredentialDialog
        modalOpen={modalOpen}
        setModalOpen={setModalOpen}
        currentProvider={provider}
      />
    </ProviderCredentialModalContext.Provider>
  );
};

export const useProviderCredentialModal = () =>
  useContext(ProviderCredentialModalContext);
