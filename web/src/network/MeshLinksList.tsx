import { DataTable, EditButton, List } from "react-admin";

export const MeshLinksList = () => {
        return <List>
            <DataTable bulkActionButtons={false}>
            <DataTable.Col source="from" render={record => "0x" + record.from.toString(16).toUpperCase()} />
            <DataTable.Col source="to" render={record => "0x" + record.to.toString(16).toUpperCase()} />
            <DataTable.Col source="description" />
            <DataTable.NumberCol label="Cost" source="weight" options={{ style: "percent" }} />
            <DataTable.Col>
                <EditButton />
            </DataTable.Col>
        </DataTable>
    </List>;
};