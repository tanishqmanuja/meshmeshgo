import { Card, CardContent, Grid, Typography } from "@mui/material";
import { Button, CreateButton, DataTable, List, Title, useCreate, useGetOne } from "react-admin";


export const Discovery = () => {
    const { isPending, error, data: discovery } = useGetOne('neighbors/discovery', { id: 0, meta: { discovery: true }}, { refetchInterval: 2500 });
    

    const StartDiscoveryButton = () => {
        const [create, { isPending }] = useCreate('neighbors/discovery', {data: {}});
        const startDiscoveryHandler = () => {
            console.log('Starting discovery');
            create();
        };
        return <Button variant="contained" color="primary" disabled={isPending} label="Start discovery" onClick={startDiscoveryHandler} />
    };

    return (
        <Card>
            <Title title="Discovery" />
            <CardContent>
                <Grid container spacing={2}>
                    <Grid size={2}>
                        <StartDiscoveryButton />
                    </Grid>
                    <Grid size={10}>
                        <Typography variant="h6">Discovery: {discovery?.status}, CurrentId: {"0x" + discovery?.current_id.toString(16).toUpperCase()} Repetitions: {discovery?.repeat}</Typography>
                    </Grid>
                </Grid>
                <List resource="neighbors" queryOptions={{ refetchInterval: 2500, meta:{discovery: true}}}>
                    <DataTable>
                        <DataTable.Col source="id" />
                        <DataTable.Col source="current" />
                        <DataTable.Col source="next" />
                        <DataTable.Col source="delta" />
                    </DataTable>
                </List>
            </CardContent>
        </Card>
    );
};