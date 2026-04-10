import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { Folders } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';

export function Knowledge({ isLoading }: { isLoading?: boolean }) {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/knowledge';

  return (
    <li>
      <SidebarSimpleListItem
        navigate={currentPath}
        active={pathname === currentPath}
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <Folders size={20} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Knowledge</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
