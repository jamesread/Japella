// Service Worker for Japella
// Version: 1.0.0
const SW_VERSION = '1.0.0';
const CACHE_NAME = `japella-cache-v${SW_VERSION}`;

// Essential files to cache on install
const CACHE_FILES = [
  '/',
  '/index.html',
  '/offline.html',
  '/logo.png',
  '/manifest.json'
];

// Install event - cache essential files
self.addEventListener('install', (event) => {
  console.log('[Service Worker] Installing version', SW_VERSION);

  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('[Service Worker] Caching files');
        return cache.addAll(CACHE_FILES);
      })
      .then(() => {
        // Force the waiting service worker to become the active service worker
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('[Service Worker] Cache failed:', error);
      })
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('[Service Worker] Activating version', SW_VERSION);

  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          // Delete old caches that don't match current version
          if (cacheName !== CACHE_NAME) {
            console.log('[Service Worker] Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    })
    .then(() => {
      // Take control of all pages immediately
      return self.clients.claim();
    })
  );
});

// Fetch event - serve from cache, fallback to network
self.addEventListener('fetch', (event) => {
  // Skip non-GET requests
  if (event.request.method !== 'GET') {
    return;
  }

  const url = new URL(event.request.url);

  // Skip API requests - always go to network
  if (url.pathname.startsWith('/api/') ||
      url.pathname.startsWith('/oauth2callback') ||
      url.pathname.startsWith('/lang') ||
      url.pathname.startsWith('/upload')) {
    return;
  }

  event.respondWith(
    caches.match(event.request)
      .then((cachedResponse) => {
        // Return cached version if available
        if (cachedResponse) {
          return cachedResponse;
        }

        // Otherwise fetch from network
        return fetch(event.request)
          .then((response) => {
            // Don't cache non-successful responses or opaque responses
            if (!response || response.status !== 200 || response.type === 'opaque') {
              return response;
            }

            // Clone the response for caching
            const responseToCache = response.clone();

            caches.open(CACHE_NAME)
              .then((cache) => {
                // Only cache same-origin requests
                if (url.origin === location.origin) {
                  cache.put(event.request, responseToCache);
                }
              });

            return response;
          })
          .catch(() => {
            // If fetch fails and it's a navigation request, return offline page
            if (event.request.mode === 'navigate') {
              return caches.match('/offline.html') || new Response('Offline', {
                status: 503,
                statusText: 'Service Unavailable',
                headers: new Headers({
                  'Content-Type': 'text/html'
                })
              });
            }
            // For other requests, return a basic error
            return new Response('Network error', {
              status: 408,
              statusText: 'Request Timeout'
            });
          });
      })
  );
});

// Message event - handle version requests
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'GET_VERSION') {
    // Handle MessageChannel port
    if (event.ports && event.ports.length > 0) {
      event.ports[0].postMessage({ version: SW_VERSION });
    } else {
      // Handle direct postMessage (fallback)
      event.source.postMessage({ type: 'VERSION_RESPONSE', version: SW_VERSION }, '*');
    }
  }
});
