

document.querySelectorAll(".btn-delete").forEach(function(button) {
    button.addEventListener("click", function(event) {
        let src = event.target;
        let nodeId = src.getAttribute("data-nodeid");
        let nodeTag = src.getAttribute("data-nodetag");
        document.getElementById("nodeId").value = nodeId;
        document.getElementById("nodeTag").value = nodeTag;
        let myModal = bootstrap.Modal.getOrCreateInstance(document.getElementById('confirmDeleteModal'));
        myModal.show();
    });
});

document.querySelector(".btn-confirm-delete").addEventListener("click", function(event) {
    let myModal = bootstrap.Modal.getOrCreateInstance(document.getElementById('confirmDeleteModal'));
    myModal.hide();
});

