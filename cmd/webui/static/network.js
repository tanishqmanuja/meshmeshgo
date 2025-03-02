
document.querySelectorAll(".btn-edit-node").forEach(function(button) {
    button.addEventListener("click", function(event) {
        let src = event.target;
        let modal = bootstrap.Modal.getOrCreateInstance(document.getElementById('editNodeModal'));
        let nodeId = src.getAttribute("data-nodeid");
        let nodeTag = src.getAttribute("data-nodetag");
        let nodeInUse = src.getAttribute("data-nodeinuse");
        document.getElementById("editNodeId").value = nodeId;
        document.getElementById("editNodeTag").value = nodeTag;
        document.getElementById("editNodeInUse").checked = nodeInUse;
        modal.show();
    });
});

document.querySelector(".btn-confirm-edit-node").addEventListener("click", function(event) {
    let myModal = bootstrap.Modal.getOrCreateInstance(document.getElementById('editNodeModal'));
    myModal.hide();
    let nodeId = document.getElementById("editNodeId").value;
    let nodeTag = document.getElementById("editNodeTag").value;
    let inUse = document.getElementById("editNodeInUse").checked;
    fetch("/network/node/configure", {
        method: "POST",
        body: JSON.stringify({
            id: parseInt(nodeId),
            tag: nodeTag,
            inuse: inUse,
        }),
    });
    console.log(nodeId, nodeTag);
});


document.querySelectorAll(".btn-delete-node").forEach(function(button) {
    button.addEventListener("click", function(event) {
        let src = event.target;
        let nodeId = src.getAttribute("data-nodeid");
        let nodeTag = src.getAttribute("data-nodetag");
        document.getElementById("deleteNodeId").value = nodeId;
        document.getElementById("deleteNodeTag").value = nodeTag;
        let myModal = bootstrap.Modal.getOrCreateInstance(document.getElementById('confirmDeleteModal'));
        myModal.show();
    }); 
});

document.querySelector(".btn-confirm-delete-node").addEventListener("click", function(event) {
    let myModal = bootstrap.Modal.getOrCreateInstance(document.getElementById('confirmDeleteModal'));
    myModal.hide();
    let nodeId = document.getElementById("deleteNodeId").value;
    fetch("/network/node/delete", {
        method: "POST",
        body: JSON.stringify({
            id: parseInt(nodeId),
        }),
    });
});

