import browser from 'webextension-polyfill';
import {
  BackendPreferencesResponse,
  ExtensionConfig,
  QuickFillProfile,
} from './types';

/**
 * Storage key identifying candidate profile records in browser local storage.
 */
export const PROFILE_STORAGE_KEY = 'quick_fill_profile';

/**
 * Storage key identifying extension configuration records in browser local storage.
 */
export const CONFIGURATION_STORAGE_KEY = 'extension_configuration';

/**
 * Storage key identifying cached raw backend preferences in browser local storage.
 */
export const RAW_PREFERENCES_STORAGE_KEY = 'quick_fill_raw_preferences';

/**
 * Default candidate profile containing fallback seed values.
 */
export const DEFAULT_PROFILE: QuickFillProfile = {
  fullName: '',
  email: '',
  phone: '',
  location: '',
  country: '',
  workAuthorization: 'Legally authorized to work without sponsorship',
  visaSponsorship: 'No, I do not require visa sponsorship',
  noticePeriod: 'Immediate (available within 1-2 weeks)',
  willingToRelocate: 'Yes, open to relocation',
  skills: [],
  customLinks: [],
  customFields: [],
  deletedFieldIds: [],
  fieldOverrides: {},
};

/**
 * Default production backend URL providing authentication and user preference endpoints.
 */
export const DEFAULT_BACKEND_URL = 'https://job-cruiser-backend.duckdns.org';

/**
 * Default production web portal hosting the Flutter user interface.
 */
export const DEFAULT_WEB_PORTAL_URL = 'https://divine-beanbag-468110-v1.web.app';

/**
 * Normalizes user-supplied endpoints, resolving web application portals to the backend API.
 */
export function resolveBackendUrl(rawUrl: string): string {
  const trimmed = rawUrl.trim();
  if (!trimmed) {
    return DEFAULT_BACKEND_URL;
  }
  const cleanUrl = trimmed.replace(/\/+$/, '').replace(/\/api$/, '');
  try {
    const parsed = new URL(cleanUrl);
    if (
      parsed.hostname.includes('web.app') ||
      parsed.hostname.includes('firebaseapp.com')
    ) {
      return DEFAULT_BACKEND_URL;
    }
  } catch {
    return cleanUrl;
  }
  return cleanUrl;
}

/**
 * Default extension configuration operating in standalone local mode.
 */
export const DEFAULT_CONFIGURATION: ExtensionConfig = {
  mode: 'local',
  backendUrl: DEFAULT_BACKEND_URL,
  apiSecret: '',
  skillsDefaultMode: 'cycle_one_by_one',
};

/**
 * Service managing persistence and synchronization of user profile data and extension configuration.
 */
export class StorageService {
  /**
   * Retrieves the stored quick fill candidate profile from browser local storage.
   * Returns default profile seed values when storage is empty.
   */
  public async getProfile(): Promise<QuickFillProfile> {
    const storageResult = await browser.storage.local.get(PROFILE_STORAGE_KEY);
    const existingProfile = storageResult[PROFILE_STORAGE_KEY] as
      | QuickFillProfile
      | undefined;

    if (!existingProfile) {
      return { ...DEFAULT_PROFILE };
    }

    return {
      ...DEFAULT_PROFILE,
      ...existingProfile,
    };
  }

  /**
   * Persists the quick fill candidate profile into browser local storage only.
   */
  public async saveProfileLocally(profile: QuickFillProfile): Promise<void> {
    await browser.storage.local.set({
      [PROFILE_STORAGE_KEY]: profile,
    });
  }

  /**
   * Persists candidate profile locally and pushes updates to the Job Cruiser backend in connected mode.
   */
  public async saveProfile(
    profile: QuickFillProfile,
  ): Promise<{ success: boolean; error?: string }> {
    await this.saveProfileLocally(profile);
    const configuration = await this.getConfig();
    if (
      configuration.mode === 'connected' &&
      configuration.backendUrl.trim().length > 0 &&
      configuration.apiSecret.trim().length > 0
    ) {
      return await this.pushToBackend(profile, configuration);
    }
    return { success: true };
  }

  /**
   * Retrieves cached raw backend preferences payload from browser local storage.
   */
  public async getCachedBackendPreferences(): Promise<BackendPreferencesResponse | null> {
    const storageResult = await browser.storage.local.get(
      RAW_PREFERENCES_STORAGE_KEY,
    );
    const cached = storageResult[RAW_PREFERENCES_STORAGE_KEY] as
      | BackendPreferencesResponse
      | undefined;
    return cached ?? null;
  }

  /**
   * Persists cached raw backend preferences payload into browser local storage.
   */
  public async setCachedBackendPreferences(
    payload: BackendPreferencesResponse,
  ): Promise<void> {
    await browser.storage.local.set({
      [RAW_PREFERENCES_STORAGE_KEY]: payload,
    });
  }

  /**
   * Pushes candidate profile updates to the Job Cruiser backend API in connected mode.
   * Merges updates with existing backend preferences so unmodified fields remain intact.
   */
  public async pushToBackend(
    profile: QuickFillProfile,
    configuration?: ExtensionConfig,
  ): Promise<{ success: boolean; error?: string }> {
    const activeConfig = configuration ?? (await this.getConfig());
    if (activeConfig.mode !== 'connected' || !activeConfig.apiSecret.trim()) {
      return { success: true };
    }

    try {
      const sanitizedBaseUrl = resolveBackendUrl(activeConfig.backendUrl);
      const endpointUrl = `${sanitizedBaseUrl}/api/preferences`;

      const existingRaw = await this.getCachedBackendPreferences();

      const mergedCustomAnswers = {
        ...(existingRaw?.custom_form_answers ?? {}),
        work_authorization: profile.workAuthorization || '',
        visa_sponsorship: profile.visaSponsorship || '',
        notice_period: profile.noticePeriod || '',
        willing_to_relocate: profile.willingToRelocate || '',
        expected_salary: profile.expectedSalary || '',
        current_salary: profile.currentSalary || '',
        custom_fields: profile.customFields || [],
        field_overrides: profile.fieldOverrides || {},
        deleted_field_ids: profile.deletedFieldIds || [],
      };

      const payload: BackendPreferencesResponse = {
        ...existingRaw,
        full_name: profile.fullName || 'User',
        professional_headline: profile.professionalHeadline || '',
        email: profile.email || '',
        phone: profile.phone || '',
        location: profile.location || '',
        country: profile.country || '',
        linkedin_url: profile.linkedinUrl || '',
        github_url: profile.githubUrl || '',
        portfolio_url: profile.portfolioUrl || '',
        custom_links: profile.customLinks || [],
        skills: profile.skills || [],
        bio_summary: profile.bioSummary || '',
        bio_experience_text: profile.bioSummary || '',
        custom_form_answers: mergedCustomAnswers,
      };

      const response = await fetch(endpointUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          Authorization: `Bearer ${activeConfig.apiSecret.trim()}`,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        if (response.status === 401) {
          const updatedConfig: ExtensionConfig = {
            ...activeConfig,
            isSessionExpired: true,
          };
          await this.saveConfig(updatedConfig);
          return {
            success: false,
            error:
              'Backend request failed with status code 401: Unauthorized session token.',
          };
        }
        return {
          success: false,
          error: `Backend push failed with status code ${response.status}: ${response.statusText}`,
        };
      }

      await this.setCachedBackendPreferences(payload);
      return { success: true };
    } catch (error: unknown) {
      const failureMessage =
        error instanceof Error ? error.message : 'Failed to push profile to backend';
      return { success: false, error: failureMessage };
    }
  }

  /**
   * Retrieves the current extension configuration from browser local storage.
   * Returns default local-mode configuration when storage is uninitialized.
   */
  public async getConfig(): Promise<ExtensionConfig> {
    const storageResult = await browser.storage.local.get(
      CONFIGURATION_STORAGE_KEY,
    );
    const existingConfiguration = storageResult[CONFIGURATION_STORAGE_KEY] as
      | ExtensionConfig
      | undefined;

    if (!existingConfiguration) {
      return { ...DEFAULT_CONFIGURATION };
    }

    return {
      ...DEFAULT_CONFIGURATION,
      ...existingConfiguration,
    };
  }

  /**
   * Persists extension configuration options into browser local storage.
   */
  public async saveConfig(config: ExtensionConfig): Promise<void> {
    await browser.storage.local.set({
      [CONFIGURATION_STORAGE_KEY]: config,
    });
  }

  /**
   * Synchronizes candidate profile from the Job Cruiser backend API when connected mode is active.
   * Returns an outcome object describing whether synchronization succeeded or encountered an error.
   */
  public async syncFromBackend(): Promise<{ success: boolean; error?: string }> {
    const configuration = await this.getConfig();

    if (configuration.mode !== 'connected') {
      return {
        success: false,
        error: 'Cannot synchronize from backend while in local mode',
      };
    }

    if (!configuration.backendUrl.trim()) {
      return {
        success: false,
        error: 'Backend URL is not configured',
      };
    }

    try {
      const sanitizedBaseUrl = resolveBackendUrl(configuration.backendUrl);
      const endpointUrl = `${sanitizedBaseUrl}/api/preferences`;

      const requestHeaders: Record<string, string> = {
        Accept: 'application/json',
      };

      if (configuration.apiSecret.trim().length > 0) {
        requestHeaders.Authorization = `Bearer ${configuration.apiSecret.trim()}`;
      }

      const response = await fetch(endpointUrl, {
        method: 'GET',
        headers: requestHeaders,
      });

      if (!response.ok) {
        if (response.status === 401) {
          const updatedConfiguration: ExtensionConfig = {
            ...configuration,
            isSessionExpired: true,
          };
          await this.saveConfig(updatedConfiguration);
          return {
            success: false,
            error:
              'Backend request failed with status code 401: Unauthorized session token.',
          };
        }
        return {
          success: false,
          error: `Backend request failed with status code ${response.status}: ${response.statusText}`,
        };
      }

      const backendData = (await response.json()) as
        | BackendPreferencesResponse
        | { data?: BackendPreferencesResponse };
      const rawPayload =
        backendData && 'data' in backendData && backendData.data
          ? backendData.data
          : (backendData as BackendPreferencesResponse);

      await this.setCachedBackendPreferences(rawPayload);
      const mappedProfile = this.mapBackendResponseToProfile(rawPayload);

      await this.saveProfileLocally(mappedProfile);

      const updatedConfiguration: ExtensionConfig = {
        ...configuration,
        lastSyncedAt: Date.now(),
        isSessionExpired: false,
      };
      await this.saveConfig(updatedConfiguration);

      return {
        success: true,
      };
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Unknown synchronization error occurred';
      return {
        success: false,
        error: errorMessage,
      };
    }
  }

  /**
   * Logs into Job Cruiser using email and password, receives session JWT,
   * updates extension configuration to connected mode, and syncs the profile.
   */
  public async loginWithCredentials(
    email: string,
    password: string,
    backendUrl: string,
  ): Promise<{ success: boolean; error?: string }> {
    if (!email.trim() || !password.trim()) {
      return { success: false, error: 'Email and password are required' };
    }

    try {
      const sanitizedBaseUrl = resolveBackendUrl(backendUrl);
      const loginUrl = `${sanitizedBaseUrl}/api/login`;

      const response = await fetch(loginUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
        },
        body: JSON.stringify({
          primary_email: email.trim(),
          password: password,
        }),
      });

      if (!response.ok) {
        const errorData = (await response.json().catch(() => ({}))) as {
          error?: string;
        };
        return {
          success: false,
          error: errorData.error || `Authentication failed (status ${response.status})`,
        };
      }

      const data = (await response.json()) as { token?: string };
      if (!data.token) {
        return { success: false, error: 'No token returned from server' };
      }

      const currentConfig = await this.getConfig();
      const updatedConfig: ExtensionConfig = {
        ...currentConfig,
        mode: 'connected',
        backendUrl: sanitizedBaseUrl,
        apiSecret: data.token,
      };

      await this.saveConfig(updatedConfig);
      return await this.syncFromBackend();
    } catch (error: unknown) {
      const failureMessage =
        error instanceof Error ? error.message : 'Network error during login';
      return { success: false, error: failureMessage };
    }
  }

  /**
   * Transforms raw backend preferences response payload into the normalized QuickFillProfile model.
   * Extracts names, experiences, education entries, and custom form answers.
   */
  public mapBackendResponseToProfile(
    response:
      | BackendPreferencesResponse
      | { data?: BackendPreferencesResponse; has_preferences?: boolean },
  ): QuickFillProfile {
    const rawData =
      response && 'data' in response && response.data
        ? response.data
        : (response as BackendPreferencesResponse);

    const rawFullName = rawData.full_name?.trim() ?? '';
    let extractedFirstName: string | undefined;
    let extractedLastName: string | undefined;

    if (rawFullName.length > 0) {
      const nameParts = rawFullName.split(/\s+/);
      extractedFirstName = nameParts[0];
      if (nameParts.length > 1) {
        extractedLastName = nameParts.slice(1).join(' ');
      }
    }

    const firstExperience = rawData.experiences?.[0];
    const firstEducation = rawData.education?.[0];
    const customAnswers = rawData.custom_form_answers;

    const workAuthorization =
      customAnswers?.work_authorization &&
      customAnswers.work_authorization.trim().length > 0
        ? customAnswers.work_authorization
        : DEFAULT_PROFILE.workAuthorization;

    const visaSponsorship =
      customAnswers?.visa_sponsorship &&
      customAnswers.visa_sponsorship.trim().length > 0
        ? customAnswers.visa_sponsorship
        : DEFAULT_PROFILE.visaSponsorship;

    const noticePeriod =
      customAnswers?.notice_period &&
      customAnswers.notice_period.trim().length > 0
        ? customAnswers.notice_period
        : DEFAULT_PROFILE.noticePeriod;

    const willingToRelocate =
      customAnswers?.willing_to_relocate &&
      customAnswers.willing_to_relocate.trim().length > 0
        ? customAnswers.willing_to_relocate
        : DEFAULT_PROFILE.willingToRelocate;

    const bioSummary =
      rawData.bio_summary?.trim() ||
      rawData.bio_experience_text?.trim() ||
      undefined;

    return {
      fullName: rawFullName,
      firstName: extractedFirstName,
      lastName: extractedLastName,
      email: rawData.email ?? '',
      phone: rawData.phone ?? '',
      location: rawData.location ?? '',
      country: rawData.country ?? '',
      professionalHeadline: rawData.professional_headline,
      linkedinUrl: rawData.linkedin_url,
      githubUrl: rawData.github_url,
      portfolioUrl: rawData.portfolio_url,
      customLinks: rawData.custom_links ?? [],
      workAuthorization,
      visaSponsorship,
      noticePeriod,
      willingToRelocate,
      expectedSalary: customAnswers?.expected_salary,
      currentSalary: customAnswers?.current_salary,
      currentRole: firstExperience?.role,
      currentCompany: firstExperience?.company,
      highestDegree: firstEducation?.degree,
      institution: firstEducation?.institution,
      graduationYear: firstEducation?.year,
      skills: rawData.skills ?? [],
      bioSummary,
      customFields: customAnswers?.custom_fields ?? [],
      deletedFieldIds: customAnswers?.deleted_field_ids ?? [],
      fieldOverrides: customAnswers?.field_overrides ?? {},
    };
  }
}

/**
 * Singleton instance of StorageService for application-wide access.
 */
export const storageService = new StorageService();
