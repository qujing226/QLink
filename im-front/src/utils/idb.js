// Simple IndexedDB helper for persistent key storage
const DB_NAME = 'qlink-db'
const STORE_KEYS = 'keys'

function openDB() {
  return new Promise((resolve, reject) => {
    const req = window.indexedDB.open(DB_NAME, 1)
    req.onupgradeneeded = (event) => {
      const db = event.target.result
      if (!db.objectStoreNames.contains(STORE_KEYS)) {
        db.createObjectStore(STORE_KEYS)
      }
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export async function idbSet(key, value) {
  try {
    const db = await openDB()
    const tx = db.transaction(STORE_KEYS, 'readwrite')
    tx.objectStore(STORE_KEYS).put(value, key)
    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve(true)
      tx.onerror = () => reject(tx.error)
    })
  } catch (e) {
    console.warn('IndexedDB set failed:', e)
    return false
  }
}

export async function idbGet(key) {
  try {
    const db = await openDB()
    const tx = db.transaction(STORE_KEYS, 'readonly')
    const req = tx.objectStore(STORE_KEYS).get(key)
    return new Promise((resolve, reject) => {
      req.onsuccess = () => resolve(req.result ?? null)
      req.onerror = () => reject(req.error)
    })
  } catch (e) {
    console.warn('IndexedDB get failed:', e)
    return null
  }
}

export async function idbDelete(key) {
  try {
    const db = await openDB()
    const tx = db.transaction(STORE_KEYS, 'readwrite')
    tx.objectStore(STORE_KEYS).delete(key)
    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve(true)
      tx.onerror = () => reject(tx.error)
    })
  } catch (e) {
    console.warn('IndexedDB delete failed:', e)
    return false
  }
}

export async function idbHydrateToLocalStorage(keys) {
  try {
    const db = await openDB()
    const tx = db.transaction(STORE_KEYS, 'readonly')
    const store = tx.objectStore(STORE_KEYS)
    await Promise.all(keys.map(async (k) => {
      const value = await new Promise((resolve, reject) => {
        const r = store.get(k)
        r.onsuccess = () => resolve(r.result ?? null)
        r.onerror = () => reject(r.error)
      })
      if (value !== null && localStorage.getItem(`qlink_key_${k}`) === null) {
        localStorage.setItem(`qlink_key_${k}`, value)
      }
    }))
    return true
  } catch (e) {
    console.warn('IndexedDB hydrate failed:', e)
    return false
  }
}