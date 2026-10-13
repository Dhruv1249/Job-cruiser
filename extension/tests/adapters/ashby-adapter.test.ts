import { beforeEach, describe, expect, it } from 'vitest';
import { AshbyAdapter } from '../../lib/adapters/ashby-adapter';
import { FieldClassification } from '../../lib/types';

describe('AshbyAdapter', () => {
  let adapter: AshbyAdapter;

  beforeEach(() => {
    adapter = new AshbyAdapter();
    document.body.innerHTML = '';
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('http://localhost:3000/');
  });

  it('returns true for detect when hostname contains ashbyhq.com or jobs.ashby.io', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://jobs.ashby.io/sample-startup/role-uuid');

    expect(adapter.detect()).toBe(true);
  });

  it('enhances classification for elements inside Ashby testid containers', () => {
    const container = document.createElement('div');
    container.setAttribute('data-testid', 'application-form-email-field');
    const inputElement = document.createElement('input');
    container.appendChild(inputElement);
    document.body.appendChild(container);

    const baseClassification: FieldClassification = {
      fieldType: 'unknown',
      confidence: 0.1,
      visibleLabel: 'Email',
      inputKind: 'text',
    };

    const enhanced = adapter.enhanceClassification(inputElement, baseClassification);

    expect(enhanced.fieldType).toBe('email');
    expect(enhanced.confidence).toBe(1.0);
  });

  it('extracts metadata from Ashby page', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://jobs.ashby.io/sample/123');

    const headingElement = document.createElement('h1');
    headingElement.setAttribute('data-testid', 'job-title');
    headingElement.textContent = 'Software Architect';
    document.body.appendChild(headingElement);

    const companyElement = document.createElement('div');
    companyElement.setAttribute('data-testid', 'company-name');
    companyElement.textContent = 'Sample Cloud';
    document.body.appendChild(companyElement);

    const formElement = document.createElement('form');
    formElement.setAttribute('data-testid', 'application-form');
    document.body.appendChild(formElement);

    const metadata = adapter.getJobMetadata();

    expect(metadata).not.toBeNull();
    expect(metadata?.jobTitle).toBe('Software Architect');
    expect(metadata?.companyName).toBe('Sample Cloud');
    expect(metadata?.isApplicationPage).toBe(true);
  });
});
