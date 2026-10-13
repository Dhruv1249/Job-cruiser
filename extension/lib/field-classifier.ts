import {
  FieldClassification,
  FieldType,
  InputKind,
} from './types';

export type { FieldClassification, FieldType, InputKind };

export const FIELD_PATTERNS: Record<FieldType, RegExp[]> = {
  first_name: [/(first[_\s-]?name|given[_\s-]?name|fname|forename)/i],
  last_name: [/(last[_\s-]?name|family[_\s-]?name|lname|surname)/i],
  full_name: [/(full[_\s-]?name|applicant[_\s-]?name|legal[_\s-]?name|candidate[_\s-]?name|your[_\s-]?name|\bname\s*[*:]?$|^name\b)/i],
  email: [/(e[_\s-]?mail|email[_\s-]?addr)/i],
  phone: [/(phone|mobile|tel|cell|contact[_\s-]?number)/i],
  linkedin: [/(linkedin|linked[_\s-]?in)/i],
  github: [/(github|git[_\s-]?hub)/i],
  portfolio: [/(portfolio|personal[_\s-]?website|personal[_\s-]?url)/i],
  website: [/(website|web[_\s-]?url|homepage)/i],
  location: [/(location|city|address|current[_\s-]?location)/i],
  country: [/(country|country[_\s-]?of[_\s-]?residence)/i],
  skills: [/(skill|technolog|competenc|proficienc|tech[_\s-]?stack|expertise)/i],
  work_authorization: [/(work[_\s-]?auth|authorized[_\s-]?to[_\s-]?work|legal[_\s-]?right[_\s-]?to[_\s-]?work)/i],
  visa_sponsorship: [/(visa|sponsorship|immigration[_\s-]?status|require[_\s-]?sponsor)/i],
  notice_period: [/(notice[_\s-]?period|start[_\s-]?date|available|availability)/i],
  willing_to_relocate: [/(relocat|willing[_\s-]?to[_\s-]?move)/i],
  salary_expectation: [/(salary[_\s-]?expect|desired[_\s-]?salary|expected[_\s-]?comp)/i],
  current_salary: [/(current[_\s-]?salary|current[_\s-]?comp|present[_\s-]?salary)/i],
  degree: [/(degree|qualification|highest[_\s-]?edu)/i],
  university: [/(university|college|institution|school)/i],
  graduation_year: [/(graduation|grad[_\s-]?year|completion[_\s-]?year)/i],
  current_title: [/(current[_\s-]?title|job[_\s-]?title|position[_\s-]?title)/i],
  current_company: [/(current[_\s-]?company|current[_\s-]?employer|present[_\s-]?company)/i],
  resume_upload: [/(resume|cv|curriculum)/i],
  cover_letter_upload: [/(cover[_\s-]?letter|letter[_\s-]?of[_\s-]?intent)/i],
  years_experience: [/(years[_\s-]?of[_\s-]?exp|total[_\s-]?exp)/i],
  gpa: [/(gpa|grade[_\s-]?point)/i],
  bio_summary: [/(summary|bio|about[_\s-]?yourself|tell[_\s-]?us|describe[_\s-]?yourself)/i],
  professional_headline: [/(headline|tagline|professional[_\s-]?title)/i],
  open_ended_question: [],
  unknown: [],
};

const HTML5_AUTOCOMPLETE_MAP: Readonly<Record<string, FieldType>> = {
  'given-name': 'first_name',
  'family-name': 'last_name',
  'name': 'full_name',
  'email': 'email',
  'tel': 'phone',
  'country': 'country',
  'address-level2': 'location',
  'url': 'website',
};

interface ApplicantTrackingSystemSelectorRule {
  readonly selector: string;
  readonly fieldType: FieldType;
  readonly confidence: number;
}

const APPLICANT_TRACKING_SYSTEM_RULES: readonly ApplicantTrackingSystemSelectorRule[] = [
  {
    selector: 'input[name^="job_application[first_name"]',
    fieldType: 'first_name',
    confidence: 1.0,
  },
  {
    selector: 'input[name^="job_application[last_name"]',
    fieldType: 'last_name',
    confidence: 1.0,
  },
  {
    selector: 'input[name^="job_application[email"]',
    fieldType: 'email',
    confidence: 1.0,
  },
  {
    selector: 'input[name^="job_application[phone"]',
    fieldType: 'phone',
    confidence: 1.0,
  },
  {
    selector: '[data-automation-id*="firstName"]',
    fieldType: 'first_name',
    confidence: 1.0,
  },
  {
    selector: '[data-automation-id*="lastName"]',
    fieldType: 'last_name',
    confidence: 1.0,
  },
  {
    selector: '[data-automation-id*="email"]',
    fieldType: 'email',
    confidence: 1.0,
  },
  {
    selector: '[data-automation-id*="phone"]',
    fieldType: 'phone',
    confidence: 1.0,
  },
  {
    selector: 'input[name="name"]',
    fieldType: 'full_name',
    confidence: 0.9,
  },
  {
    selector: 'input[name="email"]',
    fieldType: 'email',
    confidence: 0.9,
  },
];

/**
 * Classifies form input elements into standardized candidate profile field categories.
 */
export class FieldClassifier {
  /**
   * Analyzes an HTML element and returns its field classification and confidence score.
   *
   * @param element - The HTML form field element to inspect.
   * @returns The classified field type, confidence level, input kind, and visible label.
   */
  public classify(element: HTMLElement): FieldClassification {
    const scores = new Map<FieldType, number>();
    const inputKind = this.determineInputKind(element);
    const visibleLabel = this.extractVisibleLabel(element);

    this.scoreByATSSpecificSelectors(element, scores);
    this.scoreByAutocompleteAttribute(element, scores);
    this.scoreByInputType(element, scores);
    this.scoreByLabelAssociation(element, scores);
    this.scoreByAttributeRegex(element, scores);

    if (inputKind === 'textarea') {
      const placeholder = element.getAttribute('placeholder') ?? '';
      const maximumDescriptionLength = Math.max(
        visibleLabel.length,
        placeholder.length
      );
      if (maximumDescriptionLength >= 50) {
        const [currentFieldType, currentHighestScore] = this.getHighestScore(scores);
        if (currentFieldType === 'unknown' || currentHighestScore < 0.5) {
          scores.set('open_ended_question', 0.7);
        }
      }
    }

    const [fieldType, confidence] = this.getHighestScore(scores);

    return {
      fieldType,
      confidence,
      inputKind,
      visibleLabel,
    };
  }

  private scoreByAutocompleteAttribute(
    element: HTMLElement,
    scores: Map<FieldType, number>
  ): void {
    const autocompleteAttribute = element.getAttribute('autocomplete');
    if (!autocompleteAttribute) {
      return;
    }

    const tokens = autocompleteAttribute.trim().toLowerCase().split(/\s+/);
    for (const token of tokens) {
      const matchingFieldType = HTML5_AUTOCOMPLETE_MAP[token];
      if (matchingFieldType) {
        const existingScore = scores.get(matchingFieldType) ?? 0;
        if (0.95 > existingScore) {
          scores.set(matchingFieldType, 0.95);
        }
      }
    }
  }

  private scoreByInputType(
    element: HTMLElement,
    scores: Map<FieldType, number>
  ): void {
    if (element.tagName.toLowerCase() !== 'input') {
      return;
    }

    const inputType = (element.getAttribute('type') ?? '').toLowerCase();
    if (inputType === 'email') {
      const existingScore = scores.get('email') ?? 0;
      if (0.9 > existingScore) {
        scores.set('email', 0.9);
      }
    } else if (inputType === 'tel') {
      const existingScore = scores.get('phone') ?? 0;
      if (0.9 > existingScore) {
        scores.set('phone', 0.9);
      }
    } else if (inputType === 'url') {
      const existingScore = scores.get('portfolio') ?? 0;
      if (0.8 > existingScore) {
        scores.set('portfolio', 0.8);
      }
    }
  }

  private scoreByLabelAssociation(
    element: HTMLElement,
    scores: Map<FieldType, number>
  ): void {
    const explicitLabel = this.extractExplicitLabel(element);
    if (!explicitLabel) {
      return;
    }

    const normalizedLabel = explicitLabel.trim();
    const inputKind = this.determineInputKind(element);

    for (const [fieldTypeKey, patterns] of Object.entries(FIELD_PATTERNS)) {
      const fieldType = fieldTypeKey as FieldType;
      if (fieldType === 'resume_upload' && inputKind !== 'file') {
        continue;
      }
      if (fieldType === 'bio_summary' && inputKind !== 'textarea') {
        continue;
      }

      for (const pattern of patterns) {
        if (pattern.test(normalizedLabel)) {
          const existingScore = scores.get(fieldType) ?? 0;
          if (0.85 > existingScore) {
            scores.set(fieldType, 0.85);
          }
          break;
        }
      }
    }
  }

  private scoreByAttributeRegex(
    element: HTMLElement,
    scores: Map<FieldType, number>
  ): void {
    const inputKind = this.determineInputKind(element);
    const identifierAttributes = ['name', 'id'];

    for (const attributeName of identifierAttributes) {
      const attributeValue = element.getAttribute(attributeName);
      if (!attributeValue) {
        continue;
      }

      for (const [fieldTypeKey, patterns] of Object.entries(FIELD_PATTERNS)) {
        const fieldType = fieldTypeKey as FieldType;
        if (fieldType === 'resume_upload' && inputKind !== 'file') {
          continue;
        }
        if (fieldType === 'bio_summary' && inputKind !== 'textarea') {
          continue;
        }

        for (const pattern of patterns) {
          if (pattern.test(attributeValue)) {
            const existingScore = scores.get(fieldType) ?? 0;
            if (0.75 > existingScore) {
              scores.set(fieldType, 0.75);
            }
            break;
          }
        }
      }
    }

    const placeholderValue = element.getAttribute('placeholder');
    if (placeholderValue) {
      for (const [fieldTypeKey, patterns] of Object.entries(FIELD_PATTERNS)) {
        const fieldType = fieldTypeKey as FieldType;
        if (fieldType === 'resume_upload' && inputKind !== 'file') {
          continue;
        }
        if (fieldType === 'bio_summary' && inputKind !== 'textarea') {
          continue;
        }

        for (const pattern of patterns) {
          if (pattern.test(placeholderValue)) {
            const existingScore = scores.get(fieldType) ?? 0;
            if (0.6 > existingScore) {
              scores.set(fieldType, 0.6);
            }
            break;
          }
        }
      }
    }
  }

  private scoreByATSSpecificSelectors(
    element: HTMLElement,
    scores: Map<FieldType, number>
  ): void {
    for (const rule of APPLICANT_TRACKING_SYSTEM_RULES) {
      try {
        if (element.matches(rule.selector)) {
          const existingScore = scores.get(rule.fieldType) ?? 0;
          if (rule.confidence > existingScore) {
            scores.set(rule.fieldType, rule.confidence);
          }
        }
      } catch {
        continue;
      }
    }
  }

  private extractExplicitLabel(element: HTMLElement): string {
    const ariaLabel = element.getAttribute('aria-label');
    if (ariaLabel && ariaLabel.trim().length > 0) {
      return ariaLabel.trim();
    }

    const ariaLabelledBy = element.getAttribute('aria-labelledby');
    if (ariaLabelledBy && element.ownerDocument) {
      const labelledByElement = element.ownerDocument.getElementById(
        ariaLabelledBy.trim()
      );
      const textContent = labelledByElement?.textContent?.trim();
      if (textContent && textContent.length > 0) {
        return textContent;
      }
    }

    if (element.id && element.ownerDocument) {
      const escapedIdentifier = element.id.replace(/["\\]/g, '\\$&');
      const associatedLabel = element.ownerDocument.querySelector(
        `label[for="${escapedIdentifier}"]`
      );
      const textContent = associatedLabel?.textContent?.trim();
      if (textContent && textContent.length > 0) {
        return textContent;
      }
    }

    const enclosingLabel = element.closest('label');
    if (enclosingLabel) {
      const clonedLabel = enclosingLabel.cloneNode(true) as HTMLElement;
      const nestedFormControls = clonedLabel.querySelectorAll(
        'input, textarea, select, button'
      );
      nestedFormControls.forEach((nestedControl) => nestedControl.remove());
      const labelText = clonedLabel.textContent?.trim();
      if (labelText && labelText.length > 0) {
        return labelText;
      }
    }

    return '';
  }

  private extractVisibleLabel(element: HTMLElement): string {
    const explicitLabel = this.extractExplicitLabel(element);
    if (explicitLabel) {
      return explicitLabel;
    }

    const placeholder = element.getAttribute('placeholder');
    if (placeholder && placeholder.trim().length > 0) {
      return placeholder.trim();
    }

    const titleAttribute = element.getAttribute('title');
    if (titleAttribute && titleAttribute.trim().length > 0) {
      return titleAttribute.trim();
    }

    return '';
  }

  private determineInputKind(element: HTMLElement): InputKind {
    const tagName = element.tagName.toLowerCase();

    if (tagName === 'textarea') {
      return 'textarea';
    }

    if (tagName === 'select') {
      const selectElement = element as HTMLSelectElement;
      return selectElement.multiple ? 'multiselect' : 'select';
    }

    const roleAttribute = element.getAttribute('role');
    const ariaHasPopup = element.getAttribute('aria-haspopup');
    if (
      roleAttribute === 'combobox' ||
      (tagName === 'div' && ariaHasPopup === 'listbox')
    ) {
      return 'combobox';
    }

    if (tagName === 'input') {
      const inputType = (element.getAttribute('type') ?? 'text').toLowerCase();
      if (inputType === 'file') {
        return 'file';
      }
      if (inputType === 'checkbox') {
        return 'checkbox';
      }
      if (inputType === 'radio') {
        return 'radio';
      }
    }

    return 'text';
  }

  private getHighestScore(
    scores: Map<FieldType, number>
  ): [FieldType, number] {
    let highestFieldType: FieldType = 'unknown';
    let highestScore = 0.0;

    for (const [fieldType, score] of scores.entries()) {
      if (score > highestScore) {
        highestScore = score;
        highestFieldType = fieldType;
      }
    }

    return [highestFieldType, highestScore];
  }
}
