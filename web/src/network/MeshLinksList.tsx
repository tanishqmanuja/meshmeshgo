import { DataTable, List } from "react-admin";

export const MeshLinksList = () => {
        return <List>
            <DataTable bulkActionButtons={false}>
                <DataTable.Col source="from"  />
                <DataTable.Col source="to" />
                <DataTable.Col source="description" />
                <DataTable.NumberCol label="Cost" source="weight" options={{ style: "percent" }} />
            </DataTable>
        </List>;
    };
