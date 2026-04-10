import { SidebarIconWrapper } from '@/app/components/navigation/sidebar/sidebar-icon-wrapper';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { SidebarSimpleListItem } from '@/app/components/navigation/sidebar/sidebar-simple-list-item';
import { Plug } from '@carbon/icons-react';
import { useLocation } from 'react-router-dom';

export function ExternalTool({ isLoading }: { isLoading?: boolean }) {
  const location = useLocation();
  const { pathname } = location;
  return (
    <li>
      <SidebarSimpleListItem
        navigate="/integration/models"
        active={pathname.includes('/integration/models')}
        loading={isLoading}
      >
        <SidebarIconWrapper>
          <Plug size={20} />
        </SidebarIconWrapper>
        <SidebarLabel isLoading={isLoading}>External integrations</SidebarLabel>
      </SidebarSimpleListItem>
    </li>
  );
}
