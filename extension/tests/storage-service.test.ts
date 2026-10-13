import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  DEFAULT_BACKEND_URL,
  DEFAULT_CONFIGURATION,
  DEFAULT_PROFILE,
  StorageService,
} from '../lib/storage-service';
import {
  BackendPreferencesResponse,
  ExtensionConfig,
  QuickFillProfile,
} from '../lib/types';

const { mockStorageState } = vi.hoisted(() => {
  const mockStorageState: { data: Record<string, unknown> } = {
    data: {},
  };
  return { mockStorageState };
});

vi.mock('webextension-polyfill', () => {
  return {
    default: {
      storage: {
        local: {
          get: vi.fn(async (keys?: string | string[] | Record<string, unknown> | null) => {
            if (typeof keys === 'string') {
              return { [keys]: mockStorageState.data[keys] };
            }
            if (Array.isArray(keys)) {
              const matchedEntries: Record<string, unknown> = {};
              for (const key of keys) {
                matchedEntries[key] = mockStorageState.data[key];
              }
              return matchedEntries;
            }
            return { ...mockStorageState.data };
          }),
          set: vi.fn(async (items: Record<string, unknown>) => {
            Object.assign(mockStorageState.data, items);
          }),
          clear: vi.fn(async () => {
            mockStorageState.data = {};
          }),
          remove: vi.fn(async (keys: string | string[]) => {
            const keysToRemove = Array.isArray(keys) ? keys : [keys];
            for (const key of keysToRemove) {
              delete mockStorageState.data[key];
            }
          }),
        },
      },
    },
  };
});

describe('StorageService', () => {
  let service: StorageService;

  beforeEach(() => {
    mockStorageState.data = {};
    service = new StorageService();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe('getProfile', () => {
    it('returns default profile seed values when storage is empty', async () => {
      const retrievedProfile = await service.getProfile();

      expect(retrievedProfile).toEqual(DEFAULT_PROFILE);
      expect(retrievedProfile.workAuthorization).toBe(
        'Legally authorized to work without sponsorship',
      );
      expect(retrievedProfile.visaSponsorship).toBe(
        'No, I do not require visa sponsorship',
      );
      expect(retrievedProfile.noticePeriod).toBe(
        'Immediate (available within 1-2 weeks)',
      );
      expect(retrievedProfile.willingToRelocate).toBe('Yes, open to relocation');
      expect(retrievedProfile.skills).toEqual([]);
      expect(retrievedProfile.customLinks).toEqual([]);
      expect(retrievedProfile.customFields).toEqual([]);
    });

    it('returns merged profile with defaults when partial profile is stored', async () => {
      mockStorageState.data['quick_fill_profile'] = {
        fullName: 'Jane Candidate',
        email: 'jane@example.com',
      };

      const retrievedProfile = await service.getProfile();

      expect(retrievedProfile.fullName).toBe('Jane Candidate');
      expect(retrievedProfile.email).toBe('jane@example.com');
      expect(retrievedProfile.workAuthorization).toBe(
        'Legally authorized to work without sponsorship',
      );
      expect(retrievedProfile.skills).toEqual([]);
    });
  });

  describe('saveProfile', () => {
    it('persists candidate profile data into local storage', async () => {
      const candidateProfile: QuickFillProfile = {
        fullName: 'Alexander Hamilton',
        firstName: 'Alexander',
        lastName: 'Hamilton',
        email: 'alexander.hamilton@treasury.gov',
        phone: '+1 212 555 1789',
        location: 'New York, NY',
        country: 'United States',
        professionalHeadline: 'Founding Financial Architect',
        linkedinUrl: 'https://linkedin.com/in/alexanderhamilton',
        githubUrl: 'https://github.com/alexanderhamilton',
        portfolioUrl: 'https://hamilton.org',
        customLinks: [
          { label: 'Writing Archive', url: 'https://hamilton.org/writings' },
        ],
        workAuthorization: 'Citizen',
        visaSponsorship: 'No',
        noticePeriod: '2 weeks',
        willingToRelocate: 'Yes',
        expectedSalary: '$200,000',
        currentSalary: '$180,000',
        currentRole: 'Secretary of the Treasury',
        currentCompany: 'Federal Government',
        highestDegree: 'Bachelor of Arts',
        institution: 'Kings College',
        graduationYear: '1776',
        skills: ['Fiscal Policy', 'Constitutional Law', 'Speechwriting'],
        bioSummary: 'Visionary statesman and institutional builder.',
        customFields: [
          {
            id: 'clearance_level',
            label: 'Security Clearance',
            value: 'Top Secret',
            category: 'Compliance',
          },
        ],
        deletedFieldIds: [],
        fieldOverrides: {},
      };

      await service.saveProfile(candidateProfile);
      const retrievedProfile = await service.getProfile();

      expect(retrievedProfile).toEqual(candidateProfile);
      expect(mockStorageState.data['quick_fill_profile']).toEqual(
        candidateProfile,
      );
    });
  });

  describe('getConfig', () => {
    it('returns default configuration when storage is uninitialized', async () => {
      const retrievedConfiguration = await service.getConfig();

      expect(retrievedConfiguration).toEqual(DEFAULT_CONFIGURATION);
      expect(retrievedConfiguration.mode).toBe('local');
      expect(retrievedConfiguration.backendUrl).toBe(DEFAULT_BACKEND_URL);
      expect(retrievedConfiguration.apiSecret).toBe('');
      expect(retrievedConfiguration.skillsDefaultMode).toBe('cycle_one_by_one');
      expect(retrievedConfiguration.lastSyncedAt).toBeUndefined();
    });
  });

  describe('saveConfig', () => {
    it('persists extension configuration options into local storage', async () => {
      const updatedConfiguration: ExtensionConfig = {
        mode: 'connected',
        backendUrl: 'https://jobcruiser.internal',
        apiSecret: 'secret-token-key',
        skillsDefaultMode: 'dump_all',
        lastSyncedAt: 1700000000000,
      };

      await service.saveConfig(updatedConfiguration);
      const retrievedConfiguration = await service.getConfig();

      expect(retrievedConfiguration).toEqual(updatedConfiguration);
      expect(mockStorageState.data['extension_configuration']).toEqual(
        updatedConfiguration,
      );
    });
  });

  describe('syncFromBackend', () => {
    it('returns an error when running in local mode', async () => {
      await service.saveConfig({
        mode: 'local',
        backendUrl: 'https://jobcruiser.internal',
        apiSecret: 'secret-token',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      const synchronizationOutcome = await service.syncFromBackend();

      expect(synchronizationOutcome.success).toBe(false);
      expect(synchronizationOutcome.error).toContain('local mode');
    });

    it('returns an error when backend URL is missing in connected mode', async () => {
      await service.saveConfig({
        mode: 'connected',
        backendUrl: '   ',
        apiSecret: 'secret-token',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      const synchronizationOutcome = await service.syncFromBackend();

      expect(synchronizationOutcome.success).toBe(false);
      expect(synchronizationOutcome.error).toBe(
        'Backend URL is not configured',
      );
    });

    it('fetches preferences, saves profile, and updates timestamp upon success', async () => {
      await service.saveConfig({
        mode: 'connected',
        backendUrl: 'https://api.jobcruiser.internal/',
        apiSecret: 'bearer-authentication-token',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      const mockBackendPayload: BackendPreferencesResponse = {
        full_name: 'Grace Brewster Murray Hopper',
        email: 'grace.hopper@navy.mil',
        phone: '+1 703 555 1944',
        location: 'Arlington, VA',
        country: 'United States',
        professional_headline: 'Rear Admiral & Pioneer Computer Scientist',
        linkedin_url: 'https://linkedin.com/in/gracehopper',
        github_url: 'https://github.com/gracehopper',
        portfolio_url: 'https://gracehopper.org',
        custom_links: [{ label: 'Patents', url: 'https://patents.google.com' }],
        skills: ['Compilers', 'COBOL', 'Machine Architecture'],
        bio_summary: 'Pioneered machine-independent programming languages.',
        experiences: [
          {
            company: 'United States Navy',
            role: 'Rear Admiral',
            duration: '1943 - 1986',
          },
        ],
        education: [
          {
            institution: 'Yale University',
            degree: 'Doctor of Philosophy in Mathematics',
            year: '1934',
            grade: 'Honors',
          },
        ],
        custom_form_answers: {
          work_authorization: 'US Citizen',
          visa_sponsorship: 'No sponsorship needed',
          notice_period: 'Available immediately',
          willing_to_relocate: 'Yes, worldwide',
          expected_salary: '$250,000',
          current_salary: '$220,000',
          custom_fields: [
            {
              id: 'military_branch',
              label: 'Military Branch',
              value: 'Navy',
              category: 'Background',
            },
          ],
        },
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        statusText: 'OK',
        json: async () => mockBackendPayload,
      });
      vi.stubGlobal('fetch', mockFetch);

      const synchronizationOutcome = await service.syncFromBackend();

      expect(synchronizationOutcome.success).toBe(true);
      expect(synchronizationOutcome.error).toBeUndefined();

      expect(mockFetch).toHaveBeenCalledWith(
        'https://api.jobcruiser.internal/api/preferences',
        {
          method: 'GET',
          headers: {
            Accept: 'application/json',
            Authorization: 'Bearer bearer-authentication-token',
          },
        },
      );

      const persistedProfile = await service.getProfile();
      expect(persistedProfile.fullName).toBe('Grace Brewster Murray Hopper');
      expect(persistedProfile.firstName).toBe('Grace');
      expect(persistedProfile.lastName).toBe('Brewster Murray Hopper');
      expect(persistedProfile.currentRole).toBe('Rear Admiral');
      expect(persistedProfile.currentCompany).toBe('United States Navy');
      expect(persistedProfile.highestDegree).toBe(
        'Doctor of Philosophy in Mathematics',
      );
      expect(persistedProfile.institution).toBe('Yale University');
      expect(persistedProfile.graduationYear).toBe('1934');
      expect(persistedProfile.customFields).toHaveLength(1);

      const persistedConfiguration = await service.getConfig();
      expect(persistedConfiguration.lastSyncedAt).toBeTypeOf('number');
      expect(persistedConfiguration.lastSyncedAt).toBeGreaterThan(0);
    });

    it('returns an error when backend request fails with non-200 status', async () => {
      await service.saveConfig({
        mode: 'connected',
        backendUrl: 'https://api.jobcruiser.internal',
        apiSecret: 'invalid-token',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
      });
      vi.stubGlobal('fetch', mockFetch);

      const synchronizationOutcome = await service.syncFromBackend();

      expect(synchronizationOutcome.success).toBe(false);
      expect(synchronizationOutcome.error).toContain('401');
      expect(synchronizationOutcome.error).toContain('Unauthorized');
    });

    it('returns an error when fetch rejects due to network failure', async () => {
      await service.saveConfig({
        mode: 'connected',
        backendUrl: 'https://api.jobcruiser.internal',
        apiSecret: 'valid-token',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      const mockFetch = vi
        .fn()
        .mockRejectedValue(new Error('Connection refused'));
      vi.stubGlobal('fetch', mockFetch);

      const synchronizationOutcome = await service.syncFromBackend();

      expect(synchronizationOutcome.success).toBe(false);
      expect(synchronizationOutcome.error).toBe('Connection refused');
    });
  });

  describe('loginWithCredentials', () => {
    it('returns an error when email or password is empty', async () => {
      const outcome = await service.loginWithCredentials(
        '',
        'secret',
        'https://job-cruiser-backend.duckdns.org',
      );
      expect(outcome.success).toBe(false);
      expect(outcome.error).toBe('Email and password are required');
    });

    it('resolves web portal URL to DuckDNS backend and calls /api/login', async () => {
      const mockFetch = vi.fn().mockImplementation(async (url: string) => {
        if (url.includes('/api/login')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ token: 'mock-jwt-session-token' }),
          };
        }
        if (url.includes('/api/preferences')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ full_name: 'Grace Hopper' }),
          };
        }
        return { ok: false, status: 404 };
      });
      vi.stubGlobal('fetch', mockFetch);

      const outcome = await service.loginWithCredentials(
        'grace@example.com',
        'superpassword',
        'https://divine-beanbag-468110-v1.web.app/',
      );

      expect(outcome.success).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        'https://job-cruiser-backend.duckdns.org/api/login',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            primary_email: 'grace@example.com',
            password: 'superpassword',
          }),
        }),
      );

      const savedConfig = await service.getConfig();
      expect(savedConfig.mode).toBe('connected');
      expect(savedConfig.apiSecret).toBe('mock-jwt-session-token');
      expect(savedConfig.backendUrl).toBe(
        'https://job-cruiser-backend.duckdns.org',
      );
    });

    it('returns server error message upon invalid credentials', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Invalid credentials' }),
      });
      vi.stubGlobal('fetch', mockFetch);

      const outcome = await service.loginWithCredentials(
        'wrong@example.com',
        'wrongpassword',
        'https://job-cruiser-backend.duckdns.org',
      );

      expect(outcome.success).toBe(false);
      expect(outcome.error).toBe('Invalid credentials');
    });
  });

  describe('mapBackendResponseToProfile', () => {
    it('correctly maps full payload including name splitting, experiences, education, and custom answers', () => {
      const responsePayload: BackendPreferencesResponse = {
        full_name: 'Ada Augusta Lovelace',
        email: 'ada@analyticalengine.org',
        phone: '+44 20 7946 0912',
        location: 'London, England',
        country: 'United Kingdom',
        professional_headline: 'First Computer Programmer',
        linkedin_url: 'https://linkedin.com/in/adalovelace',
        github_url: 'https://github.com/adalovelace',
        portfolio_url: 'https://adalovelace.org',
        custom_links: [
          {
            label: 'Notes on the Analytical Engine',
            url: 'https://adalovelace.org/notes',
          },
        ],
        skills: ['Algorithm Design', 'Mathematics', 'Poetical Science'],
        bio_summary: 'Created the first algorithm for Charles Babbages machine.',
        experiences: [
          {
            company: 'Babbage Analytical Laboratory',
            role: 'Lead Algorithmic Researcher',
            duration: '1842 - 1852',
          },
          {
            company: 'London Mathematical Society',
            role: 'Scholar',
            duration: '1838 - 1842',
          },
        ],
        education: [
          {
            institution: 'University of London Tutorship',
            degree: 'Master of Mathematics',
            year: '1835',
            grade: 'First Class',
          },
        ],
        custom_form_answers: {
          work_authorization: 'Authorized indefinitely',
          visa_sponsorship: 'Not required',
          notice_period: 'One month',
          willing_to_relocate: 'Yes, Europe',
          expected_salary: '£150,000',
          current_salary: '£120,000',
          custom_fields: [
            {
              id: 'mentor_name',
              label: 'Primary Mentor',
              value: 'Mary Somerville',
              category: 'Academic',
            },
          ],
        },
      };

      const mappedProfile = service.mapBackendResponseToProfile(responsePayload);

      expect(mappedProfile.fullName).toBe('Ada Augusta Lovelace');
      expect(mappedProfile.firstName).toBe('Ada');
      expect(mappedProfile.lastName).toBe('Augusta Lovelace');
      expect(mappedProfile.email).toBe('ada@analyticalengine.org');
      expect(mappedProfile.phone).toBe('+44 20 7946 0912');
      expect(mappedProfile.location).toBe('London, England');
      expect(mappedProfile.country).toBe('United Kingdom');
      expect(mappedProfile.professionalHeadline).toBe(
        'First Computer Programmer',
      );
      expect(mappedProfile.linkedinUrl).toBe(
        'https://linkedin.com/in/adalovelace',
      );
      expect(mappedProfile.githubUrl).toBe('https://github.com/adalovelace');
      expect(mappedProfile.portfolioUrl).toBe('https://adalovelace.org');
      expect(mappedProfile.customLinks).toEqual([
        {
          label: 'Notes on the Analytical Engine',
          url: 'https://adalovelace.org/notes',
        },
      ]);
      expect(mappedProfile.skills).toEqual([
        'Algorithm Design',
        'Mathematics',
        'Poetical Science',
      ]);
      expect(mappedProfile.bioSummary).toBe(
        'Created the first algorithm for Charles Babbages machine.',
      );
      expect(mappedProfile.currentRole).toBe('Lead Algorithmic Researcher');
      expect(mappedProfile.currentCompany).toBe(
        'Babbage Analytical Laboratory',
      );
      expect(mappedProfile.highestDegree).toBe('Master of Mathematics');
      expect(mappedProfile.institution).toBe(
        'University of London Tutorship',
      );
      expect(mappedProfile.graduationYear).toBe('1835');
      expect(mappedProfile.workAuthorization).toBe('Authorized indefinitely');
      expect(mappedProfile.visaSponsorship).toBe('Not required');
      expect(mappedProfile.noticePeriod).toBe('One month');
      expect(mappedProfile.willingToRelocate).toBe('Yes, Europe');
      expect(mappedProfile.expectedSalary).toBe('£150,000');
      expect(mappedProfile.currentSalary).toBe('£120,000');
      expect(mappedProfile.customFields).toEqual([
        {
          id: 'mentor_name',
          label: 'Primary Mentor',
          value: 'Mary Somerville',
          category: 'Academic',
        },
      ]);
    });

    it('handles single word name without last name', () => {
      const responsePayload: BackendPreferencesResponse = {
        full_name: 'Plato',
      };

      const mappedProfile = service.mapBackendResponseToProfile(responsePayload);

      expect(mappedProfile.fullName).toBe('Plato');
      expect(mappedProfile.firstName).toBe('Plato');
      expect(mappedProfile.lastName).toBeUndefined();
    });

    it('handles empty full name gracefully', () => {
      const responsePayload: BackendPreferencesResponse = {
        full_name: '   ',
      };

      const mappedProfile = service.mapBackendResponseToProfile(responsePayload);

      expect(mappedProfile.fullName).toBe('');
      expect(mappedProfile.firstName).toBeUndefined();
      expect(mappedProfile.lastName).toBeUndefined();
    });

    it('falls back to default profile answers when custom form answers are omitted', () => {
      const emptyBackendPayload: BackendPreferencesResponse = {};

      const mappedProfile =
        service.mapBackendResponseToProfile(emptyBackendPayload);

      expect(mappedProfile.fullName).toBe('');
      expect(mappedProfile.email).toBe('');
      expect(mappedProfile.workAuthorization).toBe(
        DEFAULT_PROFILE.workAuthorization,
      );
      expect(mappedProfile.visaSponsorship).toBe(
        DEFAULT_PROFILE.visaSponsorship,
      );
      expect(mappedProfile.noticePeriod).toBe(DEFAULT_PROFILE.noticePeriod);
      expect(mappedProfile.willingToRelocate).toBe(
        DEFAULT_PROFILE.willingToRelocate,
      );
      expect(mappedProfile.currentRole).toBeUndefined();
      expect(mappedProfile.currentCompany).toBeUndefined();
      expect(mappedProfile.highestDegree).toBeUndefined();
      expect(mappedProfile.institution).toBeUndefined();
      expect(mappedProfile.graduationYear).toBeUndefined();
      expect(mappedProfile.skills).toEqual([]);
      expect(mappedProfile.customLinks).toEqual([]);
      expect(mappedProfile.customFields).toEqual([]);
    });

    it('unwraps data property when backend response is wrapped in data field', () => {
      const wrappedPayload = {
        data: {
          full_name: 'Katherine Johnson',
          email: 'kjohnson@nasa.gov',
          phone: '+15551234567',
          location: 'Hampton, VA',
          country: 'United States',
          skills: ['Orbital Mechanics', 'Trajectory Calculation', 'Fortran'],
          custom_form_answers: {
            deleted_field_ids: ['profile_headline'],
            field_overrides: {
              phone: { label: 'Direct Line' },
            },
          },
        },
        has_preferences: true,
      };

      const mapped = service.mapBackendResponseToProfile(wrappedPayload);

      expect(mapped.fullName).toBe('Katherine Johnson');
      expect(mapped.firstName).toBe('Katherine');
      expect(mapped.lastName).toBe('Johnson');
      expect(mapped.email).toBe('kjohnson@nasa.gov');
      expect(mapped.phone).toBe('+15551234567');
      expect(mapped.location).toBe('Hampton, VA');
      expect(mapped.country).toBe('United States');
      expect(mapped.skills).toEqual([
        'Orbital Mechanics',
        'Trajectory Calculation',
        'Fortran',
      ]);
      expect(mapped.deletedFieldIds).toEqual(['profile_headline']);
      expect(mapped.fieldOverrides).toEqual({
        phone: { label: 'Direct Line' },
      });
    });
  });

  describe('pushToBackend', () => {
    it('pushes updated candidate profile to backend and updates cache in connected mode', async () => {
      await service.saveConfig({
        mode: 'connected',
        backendUrl: 'https://api.jobcruiser.internal',
        apiSecret: 'test-token',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      await service.setCachedBackendPreferences({
        target_roles: ['Staff Engineer'],
        experiences: [
          { company: 'NASA', role: 'Mathematician', duration: '33 years' },
        ],
      } as any);

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        statusText: 'OK',
      });
      vi.stubGlobal('fetch', mockFetch);

      const testProfile: QuickFillProfile = {
        ...DEFAULT_PROFILE,
        fullName: 'Katherine Johnson',
        email: 'kjohnson@nasa.gov',
        phone: '+15551234567',
        deletedFieldIds: ['qa_expected_salary'],
        fieldOverrides: { phone: { label: 'Direct Line' } },
      };

      const result = await service.pushToBackend(testProfile);

      expect(result.success).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        'https://api.jobcruiser.internal/api/preferences',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            Authorization: 'Bearer test-token',
            'Content-Type': 'application/json',
          }),
        }),
      );

      const cached = await service.getCachedBackendPreferences();
      expect(cached).toBeDefined();
      expect(cached?.full_name).toBe('Katherine Johnson');
      expect(cached?.custom_form_answers?.deleted_field_ids).toEqual([
        'qa_expected_salary',
      ]);
    });

    it('returns error when backend push returns 401 Unauthorized', async () => {
      await service.saveConfig({
        mode: 'connected',
        backendUrl: 'https://api.jobcruiser.internal',
        apiSecret: 'expired-token',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
      });
      vi.stubGlobal('fetch', mockFetch);

      const testProfile: QuickFillProfile = {
        ...DEFAULT_PROFILE,
        fullName: 'Katherine Johnson',
      };

      const result = await service.pushToBackend(testProfile);

      expect(result.success).toBe(false);
      expect(result.error).toContain('401');
      expect(result.error).toContain('Unauthorized');
    });

    it('saveProfile automatically pushes to backend when connected mode is active', async () => {
      await service.saveConfig({
        mode: 'connected',
        backendUrl: 'https://api.jobcruiser.internal',
        apiSecret: 'valid-session-jwt',
        skillsDefaultMode: 'cycle_one_by_one',
      });

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        statusText: 'OK',
      });
      vi.stubGlobal('fetch', mockFetch);

      const testProfile: QuickFillProfile = {
        ...DEFAULT_PROFILE,
        fullName: 'Ada Lovelace',
        email: 'ada@computing.org',
      };

      const result = await service.saveProfile(testProfile);

      expect(result.success).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        'https://api.jobcruiser.internal/api/preferences',
        expect.objectContaining({
          method: 'POST',
        }),
      );

      const locallyStored = await service.getProfile();
      expect(locallyStored.fullName).toBe('Ada Lovelace');
    });
  });
});
