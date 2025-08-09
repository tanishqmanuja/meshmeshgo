import { Edit, SimpleForm, TextInput } from "react-admin"


export const FirmwareEdit = () => {
    return (
    <Edit>
        <SimpleForm>
            <TextInput source="id" />
        </SimpleForm>
    </Edit>
    )
}