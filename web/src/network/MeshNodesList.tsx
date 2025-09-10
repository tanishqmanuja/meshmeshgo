import { BooleanField, DataTable, EditButton, List } from "react-admin"


export const MeshNodesList = () => {
    return <List sort={{ field: "id", order: "ASC" }}>
        <DataTable bulkActionButtons={false}>
            <DataTable.Col source="node" />
            <DataTable.Col source="tag" />
            <DataTable.Col source="in_use">
                <BooleanField source="in_use" />
            </DataTable.Col>
            <DataTable.Col source="path" />
            <DataTable.Col>
                <EditButton />
            </DataTable.Col>
        </DataTable>
    </List>;
};
