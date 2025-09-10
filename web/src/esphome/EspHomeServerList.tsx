import { List, DataTable, EditButton } from "react-admin";


export const EspHomeServerList = () => {
    return <List sort={{ field: "node", order: "ASC" }}>
        <DataTable bulkActionButtons={false}>
            <DataTable.Col source="node" />
            <DataTable.Col source="address" />
            <DataTable.Col source="clients" />
        </DataTable>
    </List>;
};
