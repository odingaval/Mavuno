const DB_NAME = 'mavuno';
const DB_VERSION = 1;
let db;

function openDB() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open(DB_NAME, DB_VERSION);

        request.onupgradeneeded = function(event) {
            db = event.target.result;
            if (!db.objectStoreNames.contains("produces")) {
                db.createdObjectStore("listings", { keyPath: "id"});
            }
        };

        request.onsuccess = function(event) {
            db = event.target.result;
            resolve(db);
        };

        request.onerror = function(event) {
            reject(event.target.error);
        };
    });
}

async function saveItems(storeName, items) {
    if (!db) await openDB();
    return new Promise((reslove, reject) => {
        const tx = db.transaction(storeName, "readwrite");
        const store = tx.objectStore(storeName);
        items.forEach(item => store.put(item));
        tx.oncomplete = () => resolve();
        tx.onerror = (event) => reject(event.target.error);
    });
}