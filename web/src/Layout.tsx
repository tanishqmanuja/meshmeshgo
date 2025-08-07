import type { ReactNode } from "react";
import { Layout as RALayout, CheckForApplicationUpdate, Menu } from "react-admin";
import SearchIcon from '@mui/icons-material/Search';
import HubIcon from '@mui/icons-material/Hub';


export const MyMenu = () => (
  <Menu>
      <Menu.ResourceItem name="nodes" />
      <Menu.ResourceItem name="links" />
      <Menu.Item to="/discoverylive" primaryText="Discovery" leftIcon={<SearchIcon />} />
  </Menu>
);

export const Layout = ({ children }: { children: ReactNode }) => (
  <RALayout menu={MyMenu}>
    {children}
    <CheckForApplicationUpdate />
  </RALayout>
);
