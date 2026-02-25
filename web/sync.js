async function syncWithServer(syncRequest) {
    try {
        const response = await fetch("/api/sync", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(syncRequest),
        });

        if (!response.ok) throw new Error("Sync failed");

        const data = await response.json();
        console.log("Sync successful:", data);
        return data;
    } catch (err) {
        console.error("Error during sync:", err);
    }
}

const testSyncRequest = {
    lastSync: new Date().toISOString(),
    produces: [
        {
            id: "p1",
            version: 1,
            updatedAt: new Date().toISOString(),
            deleted: false,
            name: "Eggs",
            quality: 120
        }
    ],
    listings: [
        {
            id: "l1",
            version: 1,
            updatedAt: new Date().toISOString(),
            deleted: false,
            produceId: "p1",
            price: 2500
        }
    ]
};

syncWithServer(testSyncRequest);