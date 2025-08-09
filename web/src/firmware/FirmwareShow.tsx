import { Show, SimpleShowLayout, TextField } from "react-admin"


export const FirmwareShow = () => {
    return <Show>
        <SimpleShowLayout>
            <TextField source="id" />
        </SimpleShowLayout>
    </Show>
}