import { BooleanInput, Create, TabbedForm, TextInput } from "react-admin";
import EditNoteIcon from '@mui/icons-material/EditNote';
import { formatNodeId } from "../utils";


export const MeshNodeCreate = () => {
    return <Create mutationMode="pessimistic">
        <TabbedForm>
            <TabbedForm.Tab label="General" icon={<EditNoteIcon />} iconPosition="start" sx={{ maxWidth: '40em', minHeight: 48 }}>
                <TextInput source="id" format={v => formatNodeId(v)} />
                <TextInput source="tag" />
                <BooleanInput source="in_use" />
            </TabbedForm.Tab>
        </TabbedForm>
    </Create>;
};
