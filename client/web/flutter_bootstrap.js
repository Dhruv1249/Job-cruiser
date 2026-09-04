{{flutter_js}}
{{flutter_build_config}}

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.getRegistrations().then(function(registrations) {
    for (var i = 0; i < registrations.length; i++) {
      registrations[i].unregister();
    }
  });
}

if ('caches' in window) {
  caches.keys().then(function(keys) {
    for (var i = 0; i < keys.length; i++) {
      caches.delete(keys[i]);
    }
  });
}

_flutter.loader.load();
