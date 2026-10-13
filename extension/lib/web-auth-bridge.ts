/**
 * Module providing cryptographic decryption and session detection for companion Job Cruiser web portals.
 * Enables zero-friction single-sign-on by extracting session tokens directly from authenticated tabs.
 */

/**
 * Decrypts AES-GCM encrypted tokens produced by FlutterSecureStorage on web platforms.
 */
export async function decryptFlutterSecureStorageToken(
  publicKeyBase64: string,
  cipherPayload: string,
): Promise<string | null> {
  try {
    if (!publicKeyBase64 || !cipherPayload || !cipherPayload.includes('.')) {
      return null;
    }

    const rawKeyBytes = Uint8Array.from(atob(publicKeyBase64), (character) =>
      character.charCodeAt(0),
    );

    const cryptoSubtle =
      typeof window !== 'undefined' && window.crypto?.subtle
        ? window.crypto.subtle
        : (globalThis as unknown as { crypto?: { subtle?: SubtleCrypto } }).crypto
            ?.subtle;

    if (!cryptoSubtle) {
      return null;
    }

    const importedKey = await cryptoSubtle.importKey(
      'raw',
      rawKeyBytes,
      { name: 'AES-GCM' },
      false,
      ['decrypt'],
    );

    const [ivBase64, cipherBase64] = cipherPayload.split('.');
    if (!ivBase64 || !cipherBase64) {
      return null;
    }
    const initializationVector = Uint8Array.from(atob(ivBase64), (character) =>
      character.charCodeAt(0),
    );
    const cipherBytes = Uint8Array.from(atob(cipherBase64), (character) =>
      character.charCodeAt(0),
    );

    const decryptedBuffer = await cryptoSubtle.decrypt(
      { name: 'AES-GCM', iv: initializationVector },
      importedKey,
      cipherBytes,
    );

    return new TextDecoder().decode(decryptedBuffer);
  } catch {
    return null;
  }
}

/**
 * Scans a storage interface (localStorage or sessionStorage) for stored Job Cruiser authentication tokens.
 * Supports direct key lookups, FlutterSecureStorage encrypted keys with arbitrary prefixes, and raw JWT tokens.
 */
export async function scanStorageForAuthToken(
  storage: Storage,
): Promise<string | null> {
  if (!storage || storage.length === 0) {
    return null;
  }

  const directCandidateKeys = [
    'jobcruiser_jwt',
    'jwt_token',
    'auth_token',
    'token',
    'session_token',
  ];
  for (const directKey of directCandidateKeys) {
    const directValue = storage.getItem(directKey)?.trim();
    if (directValue && directValue.startsWith('eyJ') && directValue.length > 20) {
      return directValue;
    }
  }

  for (let index = 0; index < storage.length; index += 1) {
    const currentKey = storage.key(index);
    if (!currentKey) {
      continue;
    }

    if (currentKey.endsWith('.jwt_token')) {
      const prefix = currentKey.slice(0, -'.jwt_token'.length);
      const rawEncryptionKey = storage.getItem(prefix);
      const cipherPayload = storage.getItem(currentKey);

      if (rawEncryptionKey && cipherPayload) {
        const decryptedToken = await decryptFlutterSecureStorageToken(
          rawEncryptionKey,
          cipherPayload,
        );
        if (decryptedToken && decryptedToken.trim().length > 10) {
          return decryptedToken.trim();
        }
      }
    }
  }

  for (let index = 0; index < storage.length; index += 1) {
    const currentKey = storage.key(index);
    if (!currentKey) {
      continue;
    }
    const candidateValue = storage.getItem(currentKey)?.trim();
    if (
      candidateValue &&
      candidateValue.startsWith('eyJ') &&
      candidateValue.split('.').length === 3 &&
      candidateValue.length > 30
    ) {
      return candidateValue;
    }
  }

  return null;
}

/**
 * Inspects window localStorage and sessionStorage for existing Job Cruiser authentication tokens.
 */
export async function scanWindowForAuthToken(): Promise<string | null> {
  if (typeof window === 'undefined') {
    return null;
  }

  if (window.localStorage) {
    const localStorageToken = await scanStorageForAuthToken(window.localStorage);
    if (localStorageToken) {
      return localStorageToken;
    }
  }

  if (window.sessionStorage) {
    const sessionStorageToken = await scanStorageForAuthToken(
      window.sessionStorage,
    );
    if (sessionStorageToken) {
      return sessionStorageToken;
    }
  }

  return null;
}

/**
 * Validates whether a provided URL or current window location belongs to the Job Cruiser web portal ecosystem.
 */
export function isJobCruiserPortal(locationHref: string): boolean {
  try {
    const parsedUrl = new URL(locationHref);
    const hostname = parsedUrl.hostname.toLowerCase();
    return (
      hostname.includes('web.app') ||
      hostname.includes('firebaseapp.com') ||
      hostname.includes('jobcruiser') ||
      hostname.includes('job-cruiser') ||
      hostname.includes('duckdns.org')
    );
  } catch {
    return false;
  }
}
