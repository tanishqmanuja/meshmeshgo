export const formatNodeId = (id: any) => {
    if (typeof id === 'number') {
        return "N" + id.toString(16).toUpperCase().padStart(6, '0')
    } else if (typeof id === 'string') {
        if (id.startsWith('0x')) {
            return "N" + id.slice(2).toUpperCase().padStart(6, '0')
        }
    }
    return id
};


