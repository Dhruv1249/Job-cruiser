import { defineContentScript } from 'wxt/utils/define-content-script';
import browser from 'webextension-polyfill';
import { AutocompleteOverlay } from '../lib/autocomplete-overlay';
import { FieldClassifier } from '../lib/field-classifier';
import { FocusMonitor } from '../lib/focus-monitor';
import { InputInjector } from '../lib/input-injector';
import { SkillsInjector } from '../lib/skills-injector';
import {
  CONFIGURATION_STORAGE_KEY,
  DEFAULT_CONFIGURATION,
  DEFAULT_PROFILE,
  PROFILE_STORAGE_KEY,
} from '../lib/storage-service';
import {
  ExtensionConfig,
  ExtensionMessage,
  QuickFillProfile,
} from '../lib/types';
import {
  isJobCruiserPortal,
  scanWindowForAuthToken,
} from '../lib/web-auth-bridge';

/**
 * Job Cruiser Quick Fill content script entrypoint.
 * Automatically initializes field classification, autocomplete overlay,
 * and focus monitoring across all target web pages.
 */
export default defineContentScript({
  matches: ['<all_urls>'],
  runAt: 'document_idle',
  async main() {
    let candidateProfile: QuickFillProfile = DEFAULT_PROFILE;
    let extensionConfiguration: ExtensionConfig = DEFAULT_CONFIGURATION;

    try {
      const fetchedProfile = (await browser.runtime.sendMessage({
        action: 'GET_PROFILE',
      })) as QuickFillProfile | undefined;

      if (fetchedProfile) {
        candidateProfile = fetchedProfile;
      }
    } catch {
      candidateProfile = DEFAULT_PROFILE;
    }

    try {
      const fetchedConfiguration = (await browser.runtime.sendMessage({
        action: 'GET_CONFIG',
      })) as ExtensionConfig | undefined;

      if (fetchedConfiguration) {
        extensionConfiguration = fetchedConfiguration;
      }
    } catch {
      extensionConfiguration = DEFAULT_CONFIGURATION;
    }

    const fieldClassifier = new FieldClassifier();
    const inputInjector = new InputInjector();
    const skillsInjector = new SkillsInjector(inputInjector);
    const autocompleteOverlay = new AutocompleteOverlay(() => {});

    const focusMonitor = new FocusMonitor(
      candidateProfile,
      extensionConfiguration,
      fieldClassifier,
      autocompleteOverlay,
      inputInjector,
      skillsInjector,
    );

    focusMonitor.start();

    if (isJobCruiserPortal(window.location.href)) {
      let lastBroadcastedToken: string | null = null;

      const broadcastDetectedToken = async () => {
        const detectedToken = await scanWindowForAuthToken();
        if (detectedToken && detectedToken !== lastBroadcastedToken) {
          lastBroadcastedToken = detectedToken;
          try {
            await browser.runtime.sendMessage({
              action: 'AUTO_LOGIN_FROM_WEB',
              payload: {
                token: detectedToken,
                origin: window.location.origin,
              },
            });
          } catch {
            return;
          }
        }
      };

      void broadcastDetectedToken();

      setInterval(() => {
        void broadcastDetectedToken();
      }, 2000);

      window.addEventListener('click', () => {
        void broadcastDetectedToken();
      });

      window.addEventListener('keyup', () => {
        void broadcastDetectedToken();
      });

      window.addEventListener('storage', () => {
        void broadcastDetectedToken();
      });
    }

    browser.runtime.onMessage.addListener(async (message: unknown) => {
      const extensionMessage = message as ExtensionMessage;
      if (extensionMessage.action === 'REQUEST_WEB_TOKEN') {
        const detectedToken = await scanWindowForAuthToken();
        return { token: detectedToken };
      }
      if (extensionMessage.action === 'SAVE_PROFILE') {
        focusMonitor.stop();
        candidateProfile = extensionMessage.payload;
        focusMonitor.updateProfile(candidateProfile);
        focusMonitor.start();
        return { success: true };
      }
      return undefined;
    });

    browser.storage.onChanged.addListener((changes, areaName) => {
      if (areaName === 'local') {
        const profileChange = changes[PROFILE_STORAGE_KEY];
        if (profileChange && profileChange.newValue) {
          focusMonitor.stop();
          candidateProfile = profileChange.newValue as QuickFillProfile;
          focusMonitor.updateProfile(candidateProfile);
          focusMonitor.start();
        }

        const configurationChange = changes[CONFIGURATION_STORAGE_KEY];
        if (configurationChange && configurationChange.newValue) {
          extensionConfiguration = configurationChange.newValue as ExtensionConfig;
          focusMonitor.updateConfig(extensionConfiguration);
        }
      }
    });
  },
});
