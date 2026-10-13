import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AutocompleteOverlay } from '../lib/autocomplete-overlay';
import { FieldClassifier } from '../lib/field-classifier';
import { FocusMonitor } from '../lib/focus-monitor';
import { InputInjector } from '../lib/input-injector';
import { SkillsInjector } from '../lib/skills-injector';
import { ExtensionConfig, OverlaySuggestion, QuickFillProfile } from '../lib/types';

describe('FocusMonitor', () => {
  let focusMonitor: FocusMonitor;
  let candidateProfile: QuickFillProfile;
  let extensionConfiguration: ExtensionConfig;
  let classifier: FieldClassifier;
  let overlay: AutocompleteOverlay;
  let injector: InputInjector;
  let skillsInjector: SkillsInjector;

  beforeEach(() => {
    document.body.innerHTML = '';

    candidateProfile = {
      fullName: 'Marie Skłodowska Curie',
      firstName: 'Marie',
      lastName: 'Curie',
      email: 'marie.curie@radium.org',
      phone: '+33 1 42 68 00 00',
      location: 'Paris, France',
      country: 'France',
      linkedinUrl: 'https://linkedin.com/in/mariecurie',
      githubUrl: 'https://github.com/mariecurie',
      portfolioUrl: 'https://mariecurie.info',
      professionalHeadline: 'Physicist & Chemist | Nobel Laureate',
      currentCompany: 'Sorbonne University',
      currentRole: 'Professor of General Physics',
      highestDegree: 'Doctor of Science',
      institution: 'University of Paris',
      graduationYear: '1903',
      workAuthorization: 'Authorized to work in EU',
      visaSponsorship: 'No sponsorship required',
      noticePeriod: '30 days',
      willingToRelocate: 'Yes, worldwide',
      expectedSalary: '150000',
      currentSalary: '120000',
      bioSummary: 'Pioneered research on radioactivity.',
      skills: ['Radioactivity', 'Spectrometry', 'Nuclear Physics'],
      customLinks: [
        { label: 'Publications', url: 'https://curie.org/papers' },
      ],
      customFields: [
        { id: 'nobel_prizes', label: 'Nobel Prizes', value: 'Physics (1903), Chemistry (1911)', category: 'Honors' },
      ],
    };

    extensionConfiguration = {
      isEnabled: true,
      mode: 'local',
      backendUrl: '',
      apiSecret: '',
      skillsDefaultMode: 'cycle_one_by_one',
    };

    classifier = new FieldClassifier();
    overlay = new AutocompleteOverlay(() => {});
    injector = new InputInjector();
    skillsInjector = new SkillsInjector(injector);

    focusMonitor = new FocusMonitor(
      candidateProfile,
      extensionConfiguration,
      classifier,
      overlay,
      injector,
      skillsInjector
    );
  });

  it('shows autocomplete overlay when focused on a recognized form input', () => {
    focusMonitor.start();

    const inputElement = document.createElement('input');
    inputElement.name = 'email';
    document.body.appendChild(inputElement);

    const overlayShowSpy = vi.spyOn(overlay, 'show');

    inputElement.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));

    expect(overlayShowSpy).toHaveBeenCalledTimes(1);
    expect(overlayShowSpy).toHaveBeenCalledWith(
      inputElement,
      expect.arrayContaining([
        expect.objectContaining({
          icon: 'mail',
          value: 'marie.curie@radium.org',
        }),
      ])
    );

    focusMonitor.stop();
  });

  it('does not trigger overlay when extension is disabled in configuration', () => {
    focusMonitor.updateConfig({
      ...extensionConfiguration,
      isEnabled: false,
    });
    focusMonitor.start();

    const inputElement = document.createElement('input');
    inputElement.name = 'email';
    document.body.appendChild(inputElement);

    const overlayShowSpy = vi.spyOn(overlay, 'show');

    inputElement.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));

    expect(overlayShowSpy).not.toHaveBeenCalled();

    focusMonitor.stop();
  });

  it('ignores hidden or disabled input elements', () => {
    focusMonitor.start();

    const hiddenInput = document.createElement('input');
    hiddenInput.type = 'hidden';
    hiddenInput.name = 'email';
    document.body.appendChild(hiddenInput);

    const disabledInput = document.createElement('input');
    disabledInput.disabled = true;
    disabledInput.name = 'email';
    document.body.appendChild(disabledInput);

    const overlayShowSpy = vi.spyOn(overlay, 'show');

    hiddenInput.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));
    disabledInput.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));

    expect(overlayShowSpy).not.toHaveBeenCalled();

    focusMonitor.stop();
  });

  it('builds two skills suggestions for cycle and dump when focused on skills input', () => {
    focusMonitor.start();

    const skillsInput = document.createElement('input');
    skillsInput.name = 'skills';
    document.body.appendChild(skillsInput);

    const overlayShowSpy = vi.spyOn(overlay, 'show');

    skillsInput.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));

    expect(overlayShowSpy).toHaveBeenCalledTimes(1);
    const passedSuggestions = overlayShowSpy.mock.calls[0]?.[1] as OverlaySuggestion[];

    expect(passedSuggestions).toHaveLength(2);
    expect(passedSuggestions[0]?.action).toBe('skills_cycle');
    expect(passedSuggestions[0]?.primaryText).toBe('Cycle One-by-One');
    expect(passedSuggestions[1]?.action).toBe('skills_dump');
    expect(passedSuggestions[1]?.primaryText).toBe('Dump All');

    focusMonitor.stop();
  });

  it('matches custom fields fuzzily by visible label', () => {
    focusMonitor.start();

    const labelElement = document.createElement('label');
    labelElement.textContent = 'Nobel Prizes Won';
    const inputElement = document.createElement('input');
    labelElement.appendChild(inputElement);
    document.body.appendChild(labelElement);

    const overlayShowSpy = vi.spyOn(overlay, 'show');

    inputElement.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));

    expect(overlayShowSpy).toHaveBeenCalledTimes(1);
    const passedSuggestions = overlayShowSpy.mock.calls[0]?.[1] as OverlaySuggestion[];

    const matchedCustomField = passedSuggestions.find(
      (suggestion) => suggestion.value === 'Physics (1903), Chemistry (1911)'
    );
    expect(matchedCustomField).toBeDefined();
    expect(matchedCustomField?.secondaryText).toBe('Nobel Prizes');

    focusMonitor.stop();
  });

  it('delegates text injection to InputInjector when text suggestion is selected', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const injectTextSpy = vi.spyOn(injector, 'injectTextValue');

    const suggestion: OverlaySuggestion = {
      icon: 'user',
      primaryText: 'Marie Curie',
      value: 'Marie Curie',
      action: 'fill_text',
    };

    overlay.show(inputElement, [suggestion]);

    const rowElement = overlay['shadowRoot']?.querySelector('.suggestion-row') as HTMLElement | null;
    rowElement?.click();

    expect(injectTextSpy).toHaveBeenCalledWith(inputElement, 'Marie Curie');
  });

  it('delegates select injection to InputInjector when select suggestion is selected', () => {
    const selectElement = document.createElement('select');
    const optionElement = document.createElement('option');
    optionElement.value = 'France';
    optionElement.textContent = 'France';
    selectElement.appendChild(optionElement);
    document.body.appendChild(selectElement);

    const injectSelectSpy = vi.spyOn(injector, 'injectSelectValue');

    const suggestion: OverlaySuggestion = {
      icon: 'globe',
      primaryText: 'France',
      value: 'France',
      action: 'fill_select',
    };

    overlay.show(selectElement, [suggestion]);

    const rowElement = overlay['shadowRoot']?.querySelector('.suggestion-row') as HTMLElement | null;
    rowElement?.click();

    expect(injectSelectSpy).toHaveBeenCalledWith(selectElement, 'France');
  });

  it('stops observing and hides overlay when stop is invoked', () => {
    focusMonitor.start();

    const inputElement = document.createElement('input');
    inputElement.name = 'email';
    document.body.appendChild(inputElement);

    const overlayHideSpy = vi.spyOn(overlay, 'hide');

    focusMonitor.stop();

    expect(overlayHideSpy).toHaveBeenCalledTimes(1);

    const overlayShowSpy = vi.spyOn(overlay, 'show');
    inputElement.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));
    expect(overlayShowSpy).not.toHaveBeenCalled();
  });

  it('shows fallback profile suggestions when focusing an unclassified form input', () => {
    focusMonitor.start();

    const inputElement = document.createElement('input');
    inputElement.name = 'custom_opaque_question_input';
    document.body.appendChild(inputElement);

    const overlayShowSpy = vi.spyOn(overlay, 'show');

    inputElement.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));

    expect(overlayShowSpy).toHaveBeenCalledTimes(1);
    expect(overlayShowSpy).toHaveBeenCalledWith(
      inputElement,
      expect.arrayContaining([
        expect.objectContaining({
          primaryText: 'Marie Skłodowska Curie',
        }),
      ])
    );

    focusMonitor.stop();
  });
});
