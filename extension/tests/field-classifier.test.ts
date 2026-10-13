import { beforeEach, describe, expect, it } from 'vitest';
import { FieldClassifier } from '../lib/field-classifier';

describe('FieldClassifier', () => {
  let classifier: FieldClassifier;

  beforeEach(() => {
    classifier = new FieldClassifier();
    document.body.innerHTML = '';
  });

  it('classifies first name from autocomplete attribute', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('autocomplete', 'given-name');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('first_name');
    expect(classification.confidence).toBe(0.95);
    expect(classification.inputKind).toBe('text');
  });

  it('classifies last name from autocomplete attribute', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('autocomplete', 'family-name');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('last_name');
    expect(classification.confidence).toBe(0.95);
  });

  it('classifies email from autocomplete attribute', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('autocomplete', 'email');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('email');
    expect(classification.confidence).toBe(0.95);
  });

  it('classifies phone from autocomplete attribute', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('autocomplete', 'tel');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('phone');
    expect(classification.confidence).toBe(0.95);
  });

  it('classifies location from city autocomplete token', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('autocomplete', 'address-level2');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('location');
    expect(classification.confidence).toBe(0.95);
  });

  it('classifies country from country autocomplete token', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('autocomplete', 'country');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('country');
    expect(classification.confidence).toBe(0.95);
  });

  it('classifies full name from enclosing label text', () => {
    const labelElement = document.createElement('label');
    labelElement.textContent = 'Full Legal Name';
    const inputElement = document.createElement('input');
    labelElement.appendChild(inputElement);
    document.body.appendChild(labelElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('full_name');
    expect(classification.confidence).toBe(0.85);
    expect(classification.visibleLabel).toBe('Full Legal Name');
  });

  it('classifies linkedin from associated label using for attribute', () => {
    const labelElement = document.createElement('label');
    labelElement.setAttribute('for', 'candidate-linkedin-input');
    labelElement.textContent = 'LinkedIn Profile URL';
    const inputElement = document.createElement('input');
    inputElement.id = 'candidate-linkedin-input';
    document.body.appendChild(labelElement);
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('linkedin');
    expect(classification.confidence).toBe(0.85);
    expect(classification.visibleLabel).toBe('LinkedIn Profile URL');
  });

  it('classifies github from aria-label attribute', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('aria-label', 'GitHub Profile');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('github');
    expect(classification.confidence).toBe(0.85);
    expect(classification.visibleLabel).toBe('GitHub Profile');
  });

  it('classifies portfolio from aria-labelledby element', () => {
    const labelSpan = document.createElement('span');
    labelSpan.id = 'portfolio-label-identifier';
    labelSpan.textContent = 'Personal Portfolio Website';
    const inputElement = document.createElement('input');
    inputElement.setAttribute('aria-labelledby', 'portfolio-label-identifier');
    document.body.appendChild(labelSpan);
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('portfolio');
    expect(classification.confidence).toBe(0.85);
    expect(classification.visibleLabel).toBe('Personal Portfolio Website');
  });

  it('classifies work authorization from name attribute', () => {
    const inputElement = document.createElement('input');
    inputElement.name = 'authorized_to_work_in_us';
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('work_authorization');
    expect(classification.confidence).toBe(0.75);
  });

  it('classifies visa sponsorship from placeholder attribute', () => {
    const inputElement = document.createElement('input');
    inputElement.placeholder = 'Will you require visa sponsorship?';
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('visa_sponsorship');
    expect(classification.confidence).toBe(0.6);
  });

  it('classifies first name with highest confidence for Greenhouse ATS selector', () => {
    const inputElement = document.createElement('input');
    inputElement.name = 'job_application[first_name]';
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('first_name');
    expect(classification.confidence).toBe(1.0);
  });

  it('classifies email with highest confidence for Workday ATS selector', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('data-automation-id', 'emailInputItem');
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('email');
    expect(classification.confidence).toBe(1.0);
  });

  it('classifies full name with 0.9 confidence for Lever name selector', () => {
    const inputElement = document.createElement('input');
    inputElement.name = 'name';
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('full_name');
    expect(classification.confidence).toBe(0.9);
  });

  it('assigns higher confidence to specific ATS selectors than generic name attributes', () => {
    const genericInputElement = document.createElement('input');
    genericInputElement.name = 'phone';
    document.body.appendChild(genericInputElement);

    const atsInputElement = document.createElement('input');
    atsInputElement.name = 'job_application[phone]';
    document.body.appendChild(atsInputElement);

    const genericClassification = classifier.classify(genericInputElement);
    const atsClassification = classifier.classify(atsInputElement);

    expect(genericClassification.fieldType).toBe('phone');
    expect(atsClassification.fieldType).toBe('phone');
    expect(atsClassification.confidence).toBeGreaterThan(genericClassification.confidence);
    expect(atsClassification.confidence).toBe(1.0);
    expect(genericClassification.confidence).toBe(0.75);
  });

  it('classifies file input matching cv pattern as resume upload', () => {
    const fileInputElement = document.createElement('input');
    fileInputElement.type = 'file';
    fileInputElement.name = 'candidate_resume_cv';
    document.body.appendChild(fileInputElement);

    const classification = classifier.classify(fileInputElement);

    expect(classification.fieldType).toBe('resume_upload');
    expect(classification.inputKind).toBe('file');
  });

  it('classifies textarea matching summary as bio summary', () => {
    const textareaElement = document.createElement('textarea');
    textareaElement.name = 'candidate_bio_summary';
    document.body.appendChild(textareaElement);

    const classification = classifier.classify(textareaElement);

    expect(classification.fieldType).toBe('bio_summary');
    expect(classification.inputKind).toBe('textarea');
  });

  it('classifies long textarea without pattern match as open-ended question', () => {
    const textareaElement = document.createElement('textarea');
    textareaElement.setAttribute(
      'placeholder',
      'Please describe an impactful challenge you solved that demonstrates engineering leadership.'
    );
    document.body.appendChild(textareaElement);

    const classification = classifier.classify(textareaElement);

    expect(classification.fieldType).toBe('open_ended_question');
    expect(classification.inputKind).toBe('textarea');
    expect(classification.confidence).toBe(0.7);
  });

  it('returns unknown for unrecognizable inputs', () => {
    const inputElement = document.createElement('input');
    inputElement.name = 'unrelated_opaque_identifier_98765';
    document.body.appendChild(inputElement);

    const classification = classifier.classify(inputElement);

    expect(classification.fieldType).toBe('unknown');
    expect(classification.confidence).toBe(0.0);
    expect(classification.inputKind).toBe('text');
  });

  it('correctly identifies combobox and select input kinds', () => {
    const singleSelectElement = document.createElement('select');
    expect(classifier.classify(singleSelectElement).inputKind).toBe('select');

    const multiSelectElement = document.createElement('select');
    multiSelectElement.multiple = true;
    expect(classifier.classify(multiSelectElement).inputKind).toBe('multiselect');

    const comboboxElement = document.createElement('div');
    comboboxElement.setAttribute('role', 'combobox');
    expect(classifier.classify(comboboxElement).inputKind).toBe('combobox');
  });
});
