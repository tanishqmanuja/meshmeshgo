import fakeDataProvider from "ra-data-fakerest";
import simpleRestProvider from 'ra-data-simple-rest';

export const myFakeDataProvider = fakeDataProvider({
    nodes: [
        { id: 0, tag: 'Hello, world!' },
        { id: 1, tag: 'FooBar' },
    ],
    links: [
        { id: 0, post_id: 0, tag: 'John Doe', body: 'Sensational!' },
        { id: 1, post_id: 0, tag: 'Jane Doe', body: 'I agree' },
    ],
})

export const dataProvider = simpleRestProvider('/api/v1');