import { describe, expect, it } from 'vitest';
import {
  decryptFlutterSecureStorageToken,
  isJobCruiserPortal,
  scanStorageForAuthToken,
} from '../lib/web-auth-bridge';

describe('WebAuthBridge', () => {
  describe('isJobCruiserPortal', () => {
    it('recognizes Firebase Hosting web app domain', () => {
      expect(
        isJobCruiserPortal('https://divine-beanbag-468110-v1.web.app/#/'),
      ).toBe(true);
    });

    it('recognizes DuckDNS backend domain', () => {
      expect(
        isJobCruiserPortal('https://job-cruiser-backend.duckdns.org/api/health'),
      ).toBe(true);
    });

    it('recognizes localhost or custom domain with jobcruiser', () => {
      expect(isJobCruiserPortal('https://app.jobcruiser.com/jobs')).toBe(true);
      expect(isJobCruiserPortal('https://job-cruiser.io')).toBe(true);
    });

    it('rejects external ATS and generic job sites', () => {
      expect(isJobCruiserPortal('https://boards.greenhouse.io/stripe')).toBe(
        false,
      );
      expect(isJobCruiserPortal('https://jobs.lever.co/netflix')).toBe(false);
      expect(isJobCruiserPortal('https://google.com')).toBe(false);
    });
  });

  describe('decryptFlutterSecureStorageToken', () => {
    it('returns null for empty or invalid cipher text', async () => {
      const result = await decryptFlutterSecureStorageToken('invalid', 'invalid');
      expect(result).toBeNull();
    });

    it('decrypts AES-GCM cipher payload encrypted in FlutterSecureStorage format', async () => {
      const plainJwt = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy_user_jwt_token';

      const rawKey = new Uint8Array(32);
      crypto.getRandomValues(rawKey);
      const iv = new Uint8Array(12);
      crypto.getRandomValues(iv);

      const cryptoKey = await crypto.subtle.importKey(
        'raw',
        rawKey,
        { name: 'AES-GCM' },
        false,
        ['encrypt', 'decrypt'],
      );

      const encryptedBuffer = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv },
        cryptoKey,
        new TextEncoder().encode(plainJwt),
      );

      const publicKeyBase64 = btoa(String.fromCharCode(...rawKey));
      const ivBase64 = btoa(String.fromCharCode(...iv));
      const cipherBase64 = btoa(
        String.fromCharCode(...new Uint8Array(encryptedBuffer)),
      );
      const cipherPayload = `${ivBase64}.${cipherBase64}`;

      const decrypted = await decryptFlutterSecureStorageToken(
        publicKeyBase64,
        cipherPayload,
      );

      expect(decrypted).toBe(plainJwt);
    });
  });

  describe('scanStorageForAuthToken', () => {
    it('returns null when storage is empty', async () => {
      const mockStorage: Storage = {
        length: 0,
        clear: () => {},
        getItem: () => null,
        key: () => null,
        removeItem: () => {},
        setItem: () => {},
      };
      const token = await scanStorageForAuthToken(mockStorage);
      expect(token).toBeNull();
    });

    it('extracts direct plaintext token if available', async () => {
      const plainJwt = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.dummy_user_direct_token.signature';
      const store: Record<string, string> = {
        jobcruiser_jwt: plainJwt,
      };
      const mockStorage: Storage = {
        get length() {
          return Object.keys(store).length;
        },
        clear: () => {},
        getItem: (k: string) => store[k] ?? null,
        key: (i: number) => Object.keys(store)[i] ?? null,
        removeItem: (k: string) => {
          delete store[k];
        },
        setItem: (k: string, v: string) => {
          store[k] = v;
        },
      };

      const token = await scanStorageForAuthToken(mockStorage);
      expect(token).toBe(plainJwt);
    });

    it('extracts and decrypts token from FlutterSecureStorage default keys', async () => {
      const plainJwt = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.flutter_session_token';

      const rawKey = new Uint8Array(32);
      crypto.getRandomValues(rawKey);
      const iv = new Uint8Array(12);
      crypto.getRandomValues(iv);

      const cryptoKey = await crypto.subtle.importKey(
        'raw',
        rawKey,
        { name: 'AES-GCM' },
        false,
        ['encrypt', 'decrypt'],
      );

      const encryptedBuffer = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv },
        cryptoKey,
        new TextEncoder().encode(plainJwt),
      );

      const publicKeyBase64 = btoa(String.fromCharCode(...rawKey));
      const ivBase64 = btoa(String.fromCharCode(...iv));
      const cipherBase64 = btoa(
        String.fromCharCode(...new Uint8Array(encryptedBuffer)),
      );

      const store: Record<string, string> = {
        FlutterSecureStorage: publicKeyBase64,
        'FlutterSecureStorage.jwt_token': `${ivBase64}.${cipherBase64}`,
      };

      const mockStorage: Storage = {
        get length() {
          return Object.keys(store).length;
        },
        clear: () => {},
        getItem: (k: string) => store[k] ?? null,
        key: (i: number) => Object.keys(store)[i] ?? null,
        removeItem: (k: string) => {
          delete store[k];
        },
        setItem: (k: string, v: string) => {
          store[k] = v;
        },
      };

      const token = await scanStorageForAuthToken(mockStorage);
      expect(token).toBe(plainJwt);
    });
  });
});
