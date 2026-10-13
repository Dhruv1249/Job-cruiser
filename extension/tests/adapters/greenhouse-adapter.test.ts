import { beforeEach, describe, expect, it } from 'vitest';
import { GreenhouseAdapter } from '../../lib/adapters/greenhouse-adapter';
import { FieldClassification } from '../../lib/types';

describe('GreenhouseAdapter', () => {
  let adapter: GreenhouseAdapter;

  beforeEach(() => {
    adapter = new GreenhouseAdapter();
    document.body.innerHTML = '';
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('http://localhost:3000/');
  });

  it('returns true for detect when hostname contains greenhouse.io', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://boards.greenhouse.io/company/jobs/12345');

    expect(adapter.detect()).toBe(true);
  });

  it('returns true for detect when Greenhouse name attributes exist in DOM', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://careers.acme.corp/apply');

    const inputElement = document.createElement('input');
    inputElement.setAttribute('name', 'job_application[first_name]');
    document.body.appendChild(inputElement);

    expect(adapter.detect()).toBe(true);
  });

  it('enhances classification boosting confidence for job_application[first_name] to 1.0', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('name', 'job_application[first_name]');
    document.body.appendChild(inputElement);

    const baseClassification: FieldClassification = {
      fieldType: 'first_name',
      confidence: 0.6,
      visibleLabel: 'First Name',
      inputKind: 'text',
    };

    const enhanced = adapter.enhanceClassification(inputElement, baseClassification);

    expect(enhanced.fieldType).toBe('first_name');
    expect(enhanced.confidence).toBe(1.0);
    expect(enhanced.visibleLabel).toBe('First Name');
    expect(enhanced.inputKind).toBe('text');
  });

  it('extracts job title from DOM in getJobMetadata', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://boards.greenhouse.io/acme/jobs/999');

    const titleElement = document.createElement('h1');
    titleElement.className = 'app-title';
    titleElement.textContent = 'Senior Backend Engineer';
    document.body.appendChild(titleElement);

    const companyElement = document.createElement('div');
    companyElement.className = 'company-name';
    companyElement.textContent = 'Acme Technologies';
    document.body.appendChild(companyElement);

    const formElement = document.createElement('form');
    formElement.id = 'application_form';
    document.body.appendChild(formElement);

    const metadata = adapter.getJobMetadata();

    expect(metadata).not.toBeNull();
    expect(metadata?.jobTitle).toBe('Senior Backend Engineer');
    expect(metadata?.companyName).toBe('Acme Technologies');
    expect(metadata?.isApplicationPage).toBe(true);
    expect(metadata?.jobUrl).toBe('https://boards.greenhouse.io/acme/jobs/999');
  });
});
