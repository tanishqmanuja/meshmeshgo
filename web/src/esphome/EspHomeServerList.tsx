import { List, DataTable, EditButton } from "react-admin";
import { formatNodeId } from "../utils";


export const EspHomeServerList = () => {
    return <List sort={{ field: "node", order: "ASC" }}>
        <DataTable bulkActionButtons={false}>
            <DataTable.Col source="id" render={record => formatNodeId(record.id)} />
            <DataTable.Col source="address" />
            <DataTable.Col source="clients" />
        </DataTable>
    </List>;
};
