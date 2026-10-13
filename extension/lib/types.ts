/**
 * Custom form field entry defining an arbitrary question-answer pair.
 */
export interface CustomFormField {
  id: string;
  label: string;
  value: string;
  category: string;
}

/**
 * Compatible alias for custom form field definitions.
 */
export type CustomField = CustomFormField;

/**
 * Custom web link with a descriptive label and destination URL.
 */
export interface CustomLink {
  label: string;
  url: string;
}

/**
 * Complete candidate profile stored locally in browser storage or synchronized from the backend service.
 */
export interface QuickFillProfile {
  fullName: string;
  firstName?: string;
  lastName?: string;
  email: string;
  phone: string;
  location: string;
  country: string;
  professionalHeadline?: string;
  linkedinUrl?: string;
  githubUrl?: string;
  portfolioUrl?: string;
  customLinks: Array<{ label: string; url: string }>;
  workAuthorization: string;
  visaSponsorship: string;
  noticePeriod: string;
  willingToRelocate: string;
  expectedSalary?: string;
  currentSalary?: string;
  currentRole?: string;
  currentCompany?: string;
  highestDegree?: string;
  institution?: string;
  graduationYear?: string;
  skills: string[];
  bioSummary?: string;
  customFields: CustomFormField[];
  deletedFieldIds?: string[];
  fieldOverrides?: Record<
    string,
    { label?: string; value?: string; category?: string }
  >;
}

/**
 * Configuration options governing extension storage mode, backend synchronization, and injection behavior.
 */
export interface ExtensionConfig {
  mode: 'local' | 'connected';
  backendUrl: string;
  apiSecret: string;
  lastSyncedAt?: number;
  skillsDefaultMode: 'cycle_one_by_one' | 'dump_all';
  isEnabled?: boolean;
  autoShowOverlay?: boolean;
  isSessionExpired?: boolean;
}

/**
 * Strategy mode used when inserting candidate skills into targeted application form inputs.
 */
export type SkillsInjectionMode =
  | 'cycle'
  | 'dump'
  | 'cycle_one_by_one'
  | 'dump_all';

export const SkillsInjectionMode = {
  DUMP_ALL: 'dump_all' as const,
  CYCLE_ONE_BY_ONE: 'cycle_one_by_one' as const,
  CYCLE: 'cycle' as const,
  DUMP: 'dump' as const,
};

/**
 * Cross-context message actions passed between service worker, content scripts, and side panel.
 */
export type ExtensionMessage =
  | { action: 'GET_PROFILE' }
  | { action: 'SAVE_PROFILE'; payload: QuickFillProfile }
  | { action: 'GET_CONFIG' }
  | { action: 'SAVE_CONFIG'; payload: ExtensionConfig }
  | { action: 'SYNC_FROM_BACKEND' }
  | {
      action: 'LOGIN_WITH_CREDENTIALS';
      payload: { email: string; password: string; backendUrl: string };
    }
  | {
      action: 'AUTO_LOGIN_FROM_WEB';
      payload: { token: string; origin?: string };
    }
  | { action: 'DETECT_WEB_SESSION' }
  | { action: 'REQUEST_WEB_TOKEN' }
  | { action: 'SESSION_CONNECTED'; payload: { token: string } }
  | { action: 'GET_SKILLS' }
  | {
      action: 'INJECT_SKILLS';
      payload: { mode: SkillsInjectionMode; targetTabId: number };
    }
  | {
      action: 'INJECT_AT_CURSOR';
      payload: { text: string };
    };

/**
 * Supported autocomplete suggestion action types.
 */
export type OverlaySuggestionAction =
  | 'fill_text'
  | 'fill_select'
  | 'skills_cycle'
  | 'skills_dump';

/**
 * Autocomplete suggestion candidate displayed in the on-page floating overlay.
 */
export interface OverlaySuggestion {
  icon: string;
  primaryText: string;
  secondaryText?: string;
  value: string;
  action?: OverlaySuggestionAction;
  metadata?: Record<string, unknown>;
}

/**
 * Semantic taxonomy of recognizable job application form fields.
 */
export type FieldType =
  | 'first_name'
  | 'last_name'
  | 'full_name'
  | 'email'
  | 'phone'
  | 'location'
  | 'country'
  | 'professional_headline'
  | 'linkedin'
  | 'github'
  | 'portfolio'
  | 'website'
  | 'resume_upload'
  | 'cover_letter_upload'
  | 'current_company'
  | 'current_title'
  | 'years_experience'
  | 'degree'
  | 'university'
  | 'graduation_year'
  | 'gpa'
  | 'skills'
  | 'salary_expectation'
  | 'current_salary'
  | 'work_authorization'
  | 'visa_sponsorship'
  | 'willing_to_relocate'
  | 'notice_period'
  | 'bio_summary'
  | 'open_ended_question'
  | 'unknown';

/**
 * Supported HTML input and interactive element classifications.
 */
export type InputKind =
  | 'text'
  | 'textarea'
  | 'select'
  | 'multiselect'
  | 'combobox'
  | 'file'
  | 'checkbox'
  | 'radio';

/**
 * Result of heuristic and structural classification of a form element.
 */
export interface FieldClassification {
  fieldType: FieldType;
  confidence: number;
  visibleLabel: string;
  inputKind: InputKind;
}

/**
 * Work experience item returned from the backend preferences API.
 */
export interface BackendExperienceItem {
  company: string;
  role: string;
  duration: string;
}

/**
 * Education item returned from the backend preferences API.
 */
export interface BackendEducationItem {
  institution: string;
  degree: string;
  year: string;
  grade: string;
}

/**
 * Custom form answers container returned from the backend preferences API.
 */
export interface BackendCustomFormAnswers {
  work_authorization?: string;
  visa_sponsorship?: string;
  notice_period?: string;
  willing_to_relocate?: string;
  expected_salary?: string;
  current_salary?: string;
  custom_fields?: CustomFormField[];
  field_overrides?: Record<
    string,
    { label?: string; value?: string; category?: string }
  >;
  deleted_field_ids?: string[];
  [key: string]: unknown;
}

/**
 * Shape of preference payload returned from Job Cruiser backend endpoint GET /api/preferences.
 */
export interface BackendPreferencesResponse {
  full_name?: string;
  professional_headline?: string;
  email?: string;
  phone?: string;
  location?: string;
  country?: string;
  linkedin_url?: string;
  github_url?: string;
  portfolio_url?: string;
  custom_links?: Array<{ label: string; url: string }>;
  skills?: string[];
  bio_summary?: string;
  bio_experience_text?: string;
  experiences?: Array<{ company: string; role: string; duration: string }>;
  education?: Array<{ institution: string; degree: string; year: string; grade: string }>;
  custom_form_answers?: {
    work_authorization?: string;
    visa_sponsorship?: string;
    notice_period?: string;
    willing_to_relocate?: string;
    expected_salary?: string;
    current_salary?: string;
    custom_fields?: CustomFormField[];
    field_overrides?: Record<
      string,
      { label?: string; value?: string; category?: string }
    >;
    deleted_field_ids?: string[];
    [key: string]: unknown;
  };
}

/**
 * Result tracking outcomes of automated skills injection across input fields.
 */
export interface SkillsInjectionResult {
  readonly injectedCount: number;
  readonly skippedCount: number;
  readonly mode: SkillsInjectionMode;
}
