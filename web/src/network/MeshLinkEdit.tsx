import { Edit, NumberInput, required, SimpleForm, TextInput } from "react-admin";
import { formatNodeId } from "../utils";

export const MeshLinkEdit = () => {
    return <Edit>
        <SimpleForm>
            <TextInput source="from" format={v => formatNodeId(v)} disabled />
            <TextInput source="to" format={v => formatNodeId(v)} disabled />
            <NumberInput source="weight" validate={required()} min={0} max={100} step={5} format={v => v * 100} parse={v => v / 100} />
        </SimpleForm>
    </Edit>;
};