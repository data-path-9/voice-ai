import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { Catalog } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';

export function Project({ isLoading }: { isLoading?: boolean }) {
  const location = useLocation();
  const { pathname } = location;
  const currentPath = '/organization/projects';
  return (
    <li>
      <SidebarSimpleListItem
        navigate={currentPath}
        active={pathname.includes(currentPath)}
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <Catalog size={20} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>Projects</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
