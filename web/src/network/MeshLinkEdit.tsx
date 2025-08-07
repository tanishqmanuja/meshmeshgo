import { Edit, NumberInput, required, SimpleForm, TextInput } from "react-admin";

export const MeshLinkEdit = () => {
    return <Edit>
        <SimpleForm>
            <TextInput source="from" format={v => "0x" + v.toString(16).toUpperCase()} disabled />
            <TextInput source="to" format={v => "0x" + v.toString(16).toUpperCase()} disabled />
            <NumberInput source="weight" validate={required()} min={0} max={100} step={5} format={v => v * 100} parse={v => v / 100} />
        </SimpleForm>
    </Edit>;
};