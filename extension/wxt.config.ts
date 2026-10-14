import preact from '@preact/preset-vite';
import { defineConfig } from 'wxt';

/**
 * Build and manifest configuration for the Job Cruiser Quick Fill browser extension.
 * Configures permissions, supported host origins, Preact plugin integration,
 * and strict content security policies across Chrome and Firefox targets.
 */
export default defineConfig({
  vite: () => ({
    plugins: [preact()],
  }),
  manifest: ({ browser }) => ({
    name: 'Job Cruiser Quick Fill',
    description:
      'Password-manager-style autofill for job applications. Works standalone or synced with Job Cruiser.',
    permissions:
      browser === 'firefox'
        ? ['storage', 'activeTab', 'tabs']
        : ['storage', 'sidePanel', 'activeTab', 'tabs', 'scripting'],
    action: {
      default_title: 'Job Cruiser Quick Fill',
    },
    host_permissions: [
      '*://*.greenhouse.io/*',
      '*://*.lever.co/*',
      '*://*.ashbyhq.com/*',
      '*://*.myworkdaysite.com/*',
      '*://*.myworkdayjobs.com/*',
      '*://*.smartrecruiters.com/*',
      '<all_urls>',
    ],
    browser_specific_settings: {
      gecko: {
        id: process.env.FIREFOX_EXTENSION_ID || 'jobcruiser-quickfill@jobcruiser.app',
        strict_min_version: '109.0',
      },
    },
    content_security_policy: {
      extension_pages: "script-src 'self'; credentials-delegation 'none'; object-src 'self';",
    },
  }),
});
