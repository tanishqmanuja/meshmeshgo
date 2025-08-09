import { List, DataTable, EditButton } from "react-admin";


export const EspHomeServerList = () => {
    return <List sort={{ field: "id", order: "ASC" }}>
        <DataTable bulkActionButtons={false}>
            <DataTable.Col source="id" render={record => "0x" + record.id.toString(16).toUpperCase()} />
            <DataTable.Col source="address" />
            <DataTable.Col source="clients" />
            <DataTable.Col>
                <EditButton />
            </DataTable.Col>
        </DataTable>
    </List>;
};