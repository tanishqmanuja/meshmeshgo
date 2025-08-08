import { Edit, TextInput, BooleanInput, TabbedForm, List, DataTable, useGetRecordId, NumberInput, FormDataConsumer } from "react-admin";
import EditNoteIcon from '@mui/icons-material/EditNote';
import SettingsIcon from '@mui/icons-material/Settings';


export const MeshNodeEdit = () => {
    const recordId = useGetRecordId();
    
    return <Edit mutationMode="pessimistic" queryOptions={{refetchInterval: 2500}}>
        <TabbedForm>
            <TabbedForm.Tab label="General" icon={<EditNoteIcon />} iconPosition="start" sx={{ maxWidth: '40em', minHeight: 48 }}>
                <TextInput format={v => "0x" + (v ?? 0).toString(16).toUpperCase()} parse={v => parseInt(v, 16)} source="id" disabled />
                <TextInput source="tag" />
                <BooleanInput source="in_use" />
                <TextInput source="path" disabled />
            </TabbedForm.Tab>
            <TabbedForm.Tab label="Configuration" icon={<SettingsIcon />} iconPosition="start" sx={{ maxWidth: '40em', minHeight: 48 }}>
                <TextInput source="error" format={v => v?.length > 0 ? v : "No error"} readOnly/>
                <FormDataConsumer<{error: string}>>
                    {({formData}) => (
                        formData.error.length == 0 && 
                            <>
                                <TextInput source="dev_tag" label="Device tag" />
                                <NumberInput source="channel" min={1} max={11} step={1} label="WIFI channel" />
                                <NumberInput source="tx_power" min={-1} max={20} step={1} label="TX power" />
                                <NumberInput source="groups" min={0} max={255} step={1} label="Groups" />
                            </>
                    )}
                </FormDataConsumer>
                <TextInput source="revision" readOnly/>
                <TextInput source="binded" format={v => "0x" + v.toString(16).toUpperCase()} readOnly />
                <TextInput source="flags" readOnly />
            </TabbedForm.Tab>
            <TabbedForm.Tab label="Links">
                <List resource="links" actions={false} pagination={false} filter={{ 'any': recordId }}>
                    <DataTable storeKey="links.of.node" bulkActionButtons={false} size="small">
                        <DataTable.Col source="from"  />
                        <DataTable.Col source="to" />
                        <DataTable.Col source="weight" />
                    </DataTable>
                </List>
            </TabbedForm.Tab>
        </TabbedForm>
    </Edit>;
};