import { DataTable, List } from "react-admin";


export const EsphomeClientsList = () => {
    return <List sort={{ field: "id", order: "ASC" }}>
        <DataTable bulkActionButtons={false}>
            <DataTable.Col source="id" render={record => "0x" + record.id.toString(16).toUpperCase()} />
            <DataTable.Col source="address" />
            <DataTable.Col source="tag" />
            <DataTable.Col source="active" />
            <DataTable.Col source="handle" />
            <DataTable.Col source="sent" />
            <DataTable.Col source="received" />
            <DataTable.Col source="duration" />
            <DataTable.Col source="started" />
        </DataTable>
    </List>;
};