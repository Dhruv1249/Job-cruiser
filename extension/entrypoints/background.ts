import { defineBackground } from 'wxt/utils/define-background';
import browser from 'webextension-polyfill';
import { resolveBackendUrl, storageService } from '../lib/storage-service';
import { ExtensionConfig, ExtensionMessage } from '../lib/types';
import {
  decryptFlutterSecureStorageToken,
  isJobCruiserPortal,
} from '../lib/web-auth-bridge';

/**
 * Initializes and registers background service worker handlers for extension messages
 * and browser side panel interactions.
 */
export default defineBackground(() => {
  setupSidePanelBehavior();
  setupMessageListeners();
});

/**
 * Configures Chrome side panel to automatically open when the extension action icon is clicked.
 */
function setupSidePanelBehavior(): void {
  const chromeGlobal = (
    globalThis as unknown as {
      chrome?: {
        action?: {
          onClicked?: {
            addListener: (
              callback: (tab: { windowId?: number }) => void,
            ) => void;
          };
        };
        sidePanel?: {
          setPanelBehavior?: (options: {
            openPanelOnActionClick: boolean;
          }) => Promise<void>;
          open?: (options: { windowId: number }) => Promise<void>;
        };
      };
    }
  ).chrome;

  if (chromeGlobal?.sidePanel?.setPanelBehavior) {
    chromeGlobal.sidePanel
      .setPanelBehavior({ openPanelOnActionClick: true })
      .catch(() => {});
  }

  if (chromeGlobal?.action?.onClicked) {
    chromeGlobal.action.onClicked.addListener((tab) => {
      if (tab.windowId && chromeGlobal?.sidePanel?.open) {
        chromeGlobal.sidePanel.open({ windowId: tab.windowId }).catch(() => {});
      }
    });
  }

  const browserAny = browser as unknown as {
    sidebarAction?: {
      open?: () => Promise<void>;
    };
    browserAction?: {
      onClicked?: {
        addListener: (callback: () => void) => void;
      };
    };
  };

  if (browserAny?.browserAction?.onClicked && browserAny?.sidebarAction?.open) {
    browserAny.browserAction.onClicked.addListener(() => {
      browserAny.sidebarAction?.open?.().catch(() => {});
    });
  }
}

/**
 * Registers runtime message listener for handling profile, configuration, and synchronization requests.
 */
function setupMessageListeners(): void {
  browser.runtime.onMessage.addListener(async (message: unknown) => {
    const extensionMessage = message as ExtensionMessage;

    switch (extensionMessage.action) {
      case 'GET_PROFILE':
        return await storageService.getProfile();

      case 'SAVE_PROFILE':
        await storageService.saveProfile(extensionMessage.payload);
        return { success: true };

      case 'GET_CONFIG':
        return await storageService.getConfig();

      case 'SAVE_CONFIG':
        await storageService.saveConfig(extensionMessage.payload);
        return { success: true };

      case 'SYNC_FROM_BACKEND':
        return await storageService.syncFromBackend();

      case 'LOGIN_WITH_CREDENTIALS':
        return await storageService.loginWithCredentials(
          extensionMessage.payload.email,
          extensionMessage.payload.password,
          extensionMessage.payload.backendUrl,
        );

      case 'AUTO_LOGIN_FROM_WEB': {
        const receivedToken = extensionMessage.payload.token?.trim();
        if (!receivedToken) {
          return { success: false, error: 'Empty token payload' };
        }
        const currentConfig = await storageService.getConfig();
        if (
          currentConfig.mode === 'connected' &&
          currentConfig.apiSecret === receivedToken &&
          !currentConfig.isSessionExpired
        ) {
          return { success: true, alreadyConnected: true };
        }
        const updatedConfig: ExtensionConfig = {
          ...currentConfig,
          mode: 'connected',
          backendUrl: resolveBackendUrl(currentConfig.backendUrl),
          apiSecret: receivedToken,
        };
        await storageService.saveConfig(updatedConfig);
        const syncResult = await storageService.syncFromBackend();
        try {
          await browser.runtime.sendMessage({
            action: 'SESSION_CONNECTED',
            payload: { token: receivedToken },
          });
        } catch {
          return { success: true, syncResult };
        }
        return { success: true, syncResult };
      }

      case 'DETECT_WEB_SESSION': {
        try {
          const matchingTabs = await browser.tabs.query({});
          for (const tab of matchingTabs) {
            if (tab.id && tab.url && isJobCruiserPortal(tab.url)) {
              let extractedToken: string | null = null;

              try {
                const response = (await browser.tabs.sendMessage(tab.id, {
                  action: 'REQUEST_WEB_TOKEN',
                })) as { token?: string } | undefined;
                if (response?.token?.trim()) {
                  extractedToken = response.token.trim();
                }
              } catch {
                extractedToken = null;
              }

              if (!extractedToken) {
                try {
                  const chromeGlobal = (
                    globalThis as unknown as {
                      chrome?: {
                        scripting?: {
                          executeScript: (
                            options: unknown,
                          ) => Promise<Array<{ result: unknown }>>;
                        };
                      };
                    }
                  ).chrome;

                  if (chromeGlobal?.scripting?.executeScript) {
                    const injectionResults =
                      await chromeGlobal.scripting.executeScript({
                        target: { tabId: tab.id },
                        func: () => {
                          try {
                            for (let i = 0; i < localStorage.length; i += 1) {
                              const k = localStorage.key(i);
                              if (k && k.endsWith('.jwt_token')) {
                                const prefix = k.slice(0, -'.jwt_token'.length);
                                const rawKey = localStorage.getItem(prefix);
                                const cipher = localStorage.getItem(k);
                                if (rawKey && cipher) {
                                  return { rawKey, cipher };
                                }
                              }
                              const val = k ? localStorage.getItem(k) : null;
                              if (
                                val &&
                                val.startsWith('eyJ') &&
                                val.split('.').length === 3
                              ) {
                                return { directToken: val };
                              }
                            }
                          } catch {
                            return null;
                          }
                          return null;
                        },
                      });

                    const res = injectionResults?.[0]?.result as
                      | {
                          rawKey?: string;
                          cipher?: string;
                          directToken?: string;
                        }
                      | undefined;

                    if (res?.directToken) {
                      extractedToken = res.directToken;
                    } else if (res?.rawKey && res?.cipher) {
                      extractedToken = await decryptFlutterSecureStorageToken(
                        res.rawKey,
                        res.cipher,
                      );
                    }
                  }
                } catch {
                  extractedToken = null;
                }
              }

              if (!extractedToken && tab.id) {
                try {
                  const browserTabs = browser.tabs as unknown as {
                    executeScript?: (
                      tabId: number,
                      details: { code: string },
                    ) => Promise<unknown[]>;
                  };
                  if (browserTabs?.executeScript) {
                    const codeToRun = `(() => {
                      try {
                        for (let i = 0; i < localStorage.length; i += 1) {
                          const k = localStorage.key(i);
                          if (k && k.endsWith('.jwt_token')) {
                            const prefix = k.slice(0, -'.jwt_token'.length);
                            const rawKey = localStorage.getItem(prefix);
                            const cipher = localStorage.getItem(k);
                            if (rawKey && cipher) return JSON.stringify({ rawKey, cipher });
                          }
                          const val = k ? localStorage.getItem(k) : null;
                          if (val && val.startsWith('eyJ') && val.split('.').length === 3) {
                            return JSON.stringify({ directToken: val });
                          }
                        }
                      } catch (_) {}
                      return null;
                    })()`;
                    const evalResults = await browserTabs.executeScript(tab.id, {
                      code: codeToRun,
                    });
                    const rawJson = evalResults?.[0];
                    if (typeof rawJson === 'string') {
                      const parsed = JSON.parse(rawJson) as {
                        rawKey?: string;
                        cipher?: string;
                        directToken?: string;
                      };
                      if (parsed.directToken) {
                        extractedToken = parsed.directToken;
                      } else if (parsed.rawKey && parsed.cipher) {
                        extractedToken = await decryptFlutterSecureStorageToken(
                          parsed.rawKey,
                          parsed.cipher,
                        );
                      }
                    }
                  }
                } catch {
                  extractedToken = null;
                }
              }

              if (extractedToken && extractedToken.trim().length > 10) {
                const currentConfig = await storageService.getConfig();
                const updatedConfig: ExtensionConfig = {
                  ...currentConfig,
                  mode: 'connected',
                  backendUrl: resolveBackendUrl(currentConfig.backendUrl),
                  apiSecret: extractedToken.trim(),
                };
                await storageService.saveConfig(updatedConfig);
                await storageService.syncFromBackend();
                return { success: true, token: extractedToken.trim() };
              }
            }
          }
          return {
            success: false,
            error:
              'No active authenticated Job Cruiser tab found. Make sure you are logged in on the web portal.',
          };
        } catch (error: unknown) {
          const failureMessage =
            error instanceof Error
              ? error.message
              : 'Failed querying browser tabs';
          return { success: false, error: failureMessage };
        }
      }

      case 'GET_SKILLS': {
        const candidateProfile = await storageService.getProfile();
        return candidateProfile.skills;
      }

      case 'INJECT_SKILLS': {
        if (extensionMessage.payload?.targetTabId) {
          return await browser.tabs.sendMessage(
            extensionMessage.payload.targetTabId,
            extensionMessage,
          );
        }
        return { success: false, error: 'Target tab identifier not provided' };
      }

      default:
        return undefined;
    }
  });
}
