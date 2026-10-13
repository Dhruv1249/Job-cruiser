import { AutocompleteOverlay } from './autocomplete-overlay';
import { FieldClassifier } from './field-classifier';
import { InputInjector } from './input-injector';
import { SkillsInjector } from './skills-injector';
import {
  ExtensionConfig,
  FieldClassification,
  FieldType,
  OverlaySuggestion,
  QuickFillProfile,
} from './types';

/**
 * Monitors user focus events across web form fields and coordinates classification, suggestion rendering,
 * and automated autofill injection.
 */
export class FocusMonitor {
  private profile: QuickFillProfile;
  private configuration: ExtensionConfig;
  private readonly classifier: FieldClassifier;
  private readonly overlay: AutocompleteOverlay;
  private readonly injector: InputInjector;
  private readonly skillsInjector: SkillsInjector;
  private boundFocusInHandler: ((event: FocusEvent) => void) | null = null;
  private mutationObserver: MutationObserver | null = null;
  private isRunning: boolean = false;

  /**
   * Initializes a new instance of FocusMonitor with profile data, services, and configuration.
   *
   * @param profile - The candidate profile containing resume and personal details.
   * @param config - The extension configuration governing operational behavior.
   * @param classifier - The field classifier service analyzing form controls.
   * @param overlay - The floating autocomplete suggestion overlay component.
   * @param injector - The input injector service populating form fields.
   * @param skillsInjector - The skills injector service populating technical skills.
   */
  public constructor(
    profile: QuickFillProfile,
    config: ExtensionConfig,
    classifier: FieldClassifier,
    overlay: AutocompleteOverlay,
    injector: InputInjector,
    skillsInjector: SkillsInjector
  ) {
    this.profile = profile;
    this.configuration = config;
    this.classifier = classifier;
    this.overlay = overlay;
    this.injector = injector;
    this.skillsInjector = skillsInjector;

    this.overlay.setSelectionHandler((suggestion, targetElement) => {
      this.onOverlaySuggestionSelected(suggestion, targetElement);
    });
  }

  /**
   * Attaches global focus event listeners and mutation observers to discover and respond to active form fields.
   */
  public start(): void {
    if (this.isRunning) {
      return;
    }

    this.isRunning = true;

    this.boundFocusInHandler = (event: Event) => {
      this.handleFocusIn(event as FocusEvent);
    };

    document.addEventListener('focusin', this.boundFocusInHandler, true);
    document.addEventListener('click', this.boundFocusInHandler, true);

    if (typeof MutationObserver !== 'undefined' && document.body) {
      this.mutationObserver = new MutationObserver(() => {});
      this.mutationObserver.observe(document.body, {
        childList: true,
        subtree: true,
      });
    }
  }

  /**
   * Removes all active DOM listeners, disconnects observers, and hides open autocomplete overlays.
   */
  public stop(): void {
    if (!this.isRunning) {
      return;
    }

    this.isRunning = false;

    if (this.boundFocusInHandler) {
      document.removeEventListener('focusin', this.boundFocusInHandler, true);
      document.removeEventListener('click', this.boundFocusInHandler, true);
      this.boundFocusInHandler = null;
    }

    if (this.mutationObserver) {
      this.mutationObserver.disconnect();
      this.mutationObserver = null;
    }

    this.overlay.hide();
  }

  /**
   * Updates the candidate profile data used to generate suggestions.
   *
   * @param updatedProfile - The new candidate profile data.
   */
  public updateProfile(updatedProfile: QuickFillProfile): void {
    this.profile = updatedProfile;
  }

  /**
   * Updates the extension configuration governing autofill behavior.
   *
   * @param updatedConfiguration - The new extension configuration.
   */
  public updateConfig(updatedConfiguration: ExtensionConfig): void {
    this.configuration = updatedConfiguration;
  }

  private handleFocusIn(event: FocusEvent): void {
    if (this.configuration.isEnabled === false) {
      return;
    }

    const targetElement = event.target as Element | null;
    if (!targetElement || !this.isFormField(targetElement)) {
      return;
    }

    const classification = this.classifier.classify(targetElement);
    const suggestions = this.buildSuggestionsForClassification(classification);

    if (suggestions.length > 0) {
      this.overlay.show(targetElement, suggestions);
    } else {
      this.overlay.hide();
    }
  }

  private isFormField(element: Element): element is HTMLElement {
    if (!(element instanceof HTMLElement)) {
      return false;
    }

    const tagName = element.tagName.toLowerCase();

    if (tagName === 'input') {
      const inputType = (element.getAttribute('type') ?? 'text').toLowerCase();
      const nonInteractiveTypes = [
        'hidden',
        'submit',
        'button',
        'reset',
        'image',
      ];
      return (
        !nonInteractiveTypes.includes(inputType) &&
        !element.hasAttribute('disabled')
      );
    }

    if (tagName === 'textarea' || tagName === 'select') {
      return !element.hasAttribute('disabled');
    }

    const roleAttribute = element.getAttribute('role');
    const ariaHasPopup = element.getAttribute('aria-haspopup');
    if (
      roleAttribute === 'combobox' ||
      (tagName === 'div' && ariaHasPopup === 'listbox')
    ) {
      return true;
    }

    return false;
  }

  private buildSuggestionsForClassification(
    classification: FieldClassification
  ): OverlaySuggestion[] {
    const suggestions: OverlaySuggestion[] = [];
    const fieldType = classification.fieldType;

    if (fieldType === 'skills') {
      return this.buildSkillsSuggestions();
    }

    const contactSuggestions = this.buildContactSuggestions(fieldType);
    suggestions.push(...contactSuggestions);

    if (fieldType === 'professional_headline') {
      const value = this.profile.professionalHeadline ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'briefcase',
          primaryText: value,
          secondaryText: 'Headline',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'current_company') {
      const value = this.profile.currentCompany ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'building',
          primaryText: value,
          secondaryText: 'Current Company',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'current_title') {
      const value = this.profile.currentRole ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'briefcase',
          primaryText: value,
          secondaryText: 'Current Title',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'degree') {
      const value = this.profile.highestDegree ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'graduation-cap',
          primaryText: value,
          secondaryText: 'Highest Degree',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'university') {
      const value = this.profile.institution ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'building',
          primaryText: value,
          secondaryText: 'Institution',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'graduation_year') {
      const value = this.profile.graduationYear ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'calendar',
          primaryText: value,
          secondaryText: 'Graduation Year',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'work_authorization') {
      const value = this.profile.workAuthorization ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'check',
          primaryText: value,
          secondaryText: 'Work Authorization',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'visa_sponsorship') {
      const value = this.profile.visaSponsorship ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'clipboard',
          primaryText: value,
          secondaryText: 'Visa Sponsorship',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'notice_period') {
      const value = this.profile.noticePeriod ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'clock',
          primaryText: value,
          secondaryText: 'Notice Period',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'willing_to_relocate') {
      const value = this.profile.willingToRelocate ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'truck',
          primaryText: value,
          secondaryText: 'Relocation Preference',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'salary_expectation') {
      const value = this.profile.expectedSalary ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'dollar-sign',
          primaryText: value,
          secondaryText: 'Expected Compensation',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'current_salary') {
      const value = this.profile.currentSalary ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'dollar-sign',
          primaryText: value,
          secondaryText: 'Current Compensation',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'bio_summary') {
      const value = this.profile.bioSummary ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'file-text',
          primaryText: value.length > 50 ? `${value.slice(0, 47)}...` : value,
          secondaryText: 'Bio / Summary',
          value,
          action: 'fill_text',
        });
      }
    }

    if (
      (fieldType === 'website' || fieldType === 'portfolio') &&
      this.profile.customLinks &&
      this.profile.customLinks.length > 0
    ) {
      const maximumLinks = this.profile.customLinks.slice(0, 3);
      for (const customLink of maximumLinks) {
        if (customLink.url && customLink.url.trim().length > 0) {
          suggestions.push({
            icon: 'link',
            primaryText: customLink.label,
            secondaryText: customLink.url,
            value: customLink.url,
            action: 'fill_text',
          });
        }
      }
    }

    if (
      this.profile.customFields &&
      classification.visibleLabel.trim().length > 0
    ) {
      const normalizedVisibleLabel = classification.visibleLabel
        .trim()
        .toLowerCase();

      for (const customField of this.profile.customFields) {
        if (!customField.value || customField.value.trim().length === 0) {
          continue;
        }

        const normalizedCustomLabel = customField.label.trim().toLowerCase();
        const matchesLabel =
          normalizedVisibleLabel.includes(normalizedCustomLabel) ||
          normalizedCustomLabel.includes(normalizedVisibleLabel);

        if (matchesLabel) {
          suggestions.push({
            icon: 'tag',
            primaryText: customField.value,
            secondaryText: customField.label,
            value: customField.value,
            action: 'fill_text',
          });
        }
      }
    }

    if (classification.inputKind === 'select') {
      return suggestions
        .filter((suggestion) => suggestion.value.trim().length > 0)
        .map((suggestion) => ({
          ...suggestion,
          action: 'fill_select' as const,
        }));
    }

    return suggestions.filter(
      (suggestion) =>
        suggestion.action === 'skills_cycle' ||
        suggestion.value.trim().length > 0
    );
  }

  private buildSkillsSuggestions(): OverlaySuggestion[] {
    const skills = this.profile.skills;
    if (skills.length === 0) {
      return [];
    }

    const joinedSkills = skills.join(', ');
    const previewSkills =
      joinedSkills.length > 40
        ? `${joinedSkills.slice(0, 40)}...`
        : joinedSkills;

    return [
      {
        icon: 'rotate-cw',
        primaryText: 'Cycle One-by-One',
        secondaryText: `${skills.length} skills via Enter key`,
        action: 'skills_cycle',
        value: '',
      },
      {
        icon: 'list',
        primaryText: 'Dump All',
        secondaryText: previewSkills,
        action: 'skills_dump',
        value: joinedSkills,
      },
    ];
  }

  private buildContactSuggestions(
    fieldType: FieldType
  ): OverlaySuggestion[] {
    const suggestions: OverlaySuggestion[] = [];
    const nameParts = this.profile.fullName
      ? this.profile.fullName.trim().split(/\s+/)
      : [];

    if (fieldType === 'first_name') {
      const firstNameValue = (
        this.profile.firstName?.trim() ||
        (nameParts.length > 0 ? nameParts[0] : '') ||
        ''
      ).trim();

      if (firstNameValue) {
        suggestions.push({
          icon: 'user',
          primaryText: firstNameValue,
          secondaryText: 'First Name',
          value: firstNameValue,
          action: 'fill_text',
        });
      }

      const fullNameValue = (this.profile.fullName ?? '').trim();
      if (fullNameValue && fullNameValue !== firstNameValue) {
        suggestions.push({
          icon: 'user',
          primaryText: fullNameValue,
          secondaryText: 'Full Name',
          value: fullNameValue,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'last_name') {
      const lastNameValue = (
        this.profile.lastName?.trim() ||
        (nameParts.length > 1 ? nameParts[nameParts.length - 1] : '') ||
        ''
      ).trim();

      if (lastNameValue) {
        suggestions.push({
          icon: 'user',
          primaryText: lastNameValue,
          secondaryText: 'Last Name',
          value: lastNameValue,
          action: 'fill_text',
        });
      }

      const fullNameValue = (this.profile.fullName ?? '').trim();
      if (fullNameValue && fullNameValue !== lastNameValue) {
        suggestions.push({
          icon: 'user',
          primaryText: fullNameValue,
          secondaryText: 'Full Name',
          value: fullNameValue,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'full_name') {
      const fullNameValue = (this.profile.fullName ?? '').trim();
      if (fullNameValue) {
        suggestions.push({
          icon: 'user',
          primaryText: fullNameValue,
          secondaryText: 'Full Name',
          value: fullNameValue,
          action: 'fill_text',
        });
      }

      const firstNameValue = (
        this.profile.firstName?.trim() ||
        (nameParts.length > 0 ? nameParts[0] : '') ||
        ''
      ).trim();
      if (firstNameValue && firstNameValue !== fullNameValue) {
        suggestions.push({
          icon: 'user',
          primaryText: firstNameValue,
          secondaryText: 'First Name',
          value: firstNameValue,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'email') {
      const value = this.profile.email ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'mail',
          primaryText: value,
          secondaryText: 'Email Address',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'phone') {
      const value = this.profile.phone ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'phone',
          primaryText: value,
          secondaryText: 'Phone Number',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'location') {
      const value = this.profile.location ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'map-pin',
          primaryText: value,
          secondaryText: 'Location',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'country') {
      const value = this.profile.country ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'globe',
          primaryText: value,
          secondaryText: 'Country',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'linkedin') {
      const value = this.profile.linkedinUrl ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'link',
          primaryText: value,
          secondaryText: 'LinkedIn',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'github') {
      const value = this.profile.githubUrl ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'link',
          primaryText: value,
          secondaryText: 'GitHub',
          value,
          action: 'fill_text',
        });
      }
    } else if (fieldType === 'portfolio' || fieldType === 'website') {
      const value = this.profile.portfolioUrl ?? '';
      if (value.trim()) {
        suggestions.push({
          icon: 'globe',
          primaryText: value,
          secondaryText: 'Portfolio',
          value,
          action: 'fill_text',
        });
      }
    }

    return suggestions;
  }

  private onOverlaySuggestionSelected(
    suggestion: OverlaySuggestion,
    target: HTMLElement
  ): void {
    const action = suggestion.action ?? 'fill_text';

    if (action === 'skills_cycle') {
      void this.skillsInjector.cycleOneByOne(target, [...this.profile.skills]);
      return;
    }

    if (action === 'skills_dump') {
      this.skillsInjector.dumpAll(target, [...this.profile.skills]);
      return;
    }

    if (
      action === 'fill_select' ||
      target.tagName.toLowerCase() === 'select'
    ) {
      this.injector.injectSelectValue(
        target as HTMLSelectElement,
        suggestion.value
      );
      return;
    }

    if (
      target instanceof HTMLInputElement ||
      target instanceof HTMLTextAreaElement ||
      target.tagName.toLowerCase() === 'input' ||
      target.tagName.toLowerCase() === 'textarea'
    ) {
      this.injector.injectTextValue(
        target as HTMLInputElement | HTMLTextAreaElement,
        suggestion.value
      );
    }
  }
}
