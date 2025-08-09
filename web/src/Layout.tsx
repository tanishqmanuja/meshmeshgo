import type { ReactNode } from "react";
import { Layout as RALayout, CheckForApplicationUpdate, Menu } from "react-admin";
import SearchIcon from '@mui/icons-material/Search';


export const MyMenu = () => (
  <Menu>
      <Menu.ResourceItem name="nodes" />
      <Menu.ResourceItem name="links" />
      <Menu.ResourceItem name="esphomeServers" />
      <Menu.ResourceItem name="esphomeClients" />
      <Menu.Item to="/discoverylive" primaryText="Discovery" leftIcon={<SearchIcon />} />
  </Menu>
);

export const Layout = ({ children }: { children: ReactNode }) => (
  <RALayout menu={MyMenu}>
    {children}
    <CheckForApplicationUpdate />
  </RALayout>
);
