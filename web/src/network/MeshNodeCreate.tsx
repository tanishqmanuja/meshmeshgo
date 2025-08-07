import { BooleanInput, Create, TabbedForm, TextInput } from "react-admin";
import EditNoteIcon from '@mui/icons-material/EditNote';


export const MeshNodeCreate = () => {
    return <Create mutationMode="pessimistic">
        <TabbedForm>
            <TabbedForm.Tab label="General" icon={<EditNoteIcon />} iconPosition="start" sx={{ maxWidth: '40em', minHeight: 48 }}>
                <TextInput format={v => "0x" + (v ?? 0).toString(16).toUpperCase()} parse={v => parseInt(v, 16)} source="id" />
                <TextInput source="tag" />
                <BooleanInput source="in_use" />
            </TabbedForm.Tab>
        </TabbedForm>
    </Create>;
};