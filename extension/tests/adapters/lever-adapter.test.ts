import { beforeEach, describe, expect, it } from 'vitest';
import { LeverAdapter } from '../../lib/adapters/lever-adapter';
import { FieldClassification } from '../../lib/types';

describe('LeverAdapter', () => {
  let adapter: LeverAdapter;

  beforeEach(() => {
    adapter = new LeverAdapter();
    document.body.innerHTML = '';
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('http://localhost:3000/');
  });

  it('returns true for detect when hostname contains jobs.lever.co', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://jobs.lever.co/acme-corp/123-abc');

    expect(adapter.detect()).toBe(true);
  });

  it('enhances classification for Lever name input to full_name with 1.0 confidence', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('name', 'name');
    document.body.appendChild(inputElement);

    const baseClassification: FieldClassification = {
      fieldType: 'unknown',
      confidence: 0.1,
      visibleLabel: 'Name',
      inputKind: 'text',
    };

    const enhanced = adapter.enhanceClassification(inputElement, baseClassification);

    expect(enhanced.fieldType).toBe('full_name');
    expect(enhanced.confidence).toBe(1.0);
  });

  it('enhances classification for Lever email input to email with 1.0 confidence', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('name', 'email');
    document.body.appendChild(inputElement);

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

  it('enhances classification for Lever org input to current_company with 1.0 confidence', () => {
    const inputElement = document.createElement('input');
    inputElement.setAttribute('name', 'org');
    document.body.appendChild(inputElement);

    const baseClassification: FieldClassification = {
      fieldType: 'unknown',
      confidence: 0.1,
      visibleLabel: 'Current Company',
      inputKind: 'text',
    };

    const enhanced = adapter.enhanceClassification(inputElement, baseClassification);

    expect(enhanced.fieldType).toBe('current_company');
    expect(enhanced.confidence).toBe(1.0);
  });

  it('extracts job metadata including title and company name from Lever DOM', () => {
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('https://jobs.lever.co/acme/position-uuid');

    const jobTitleContainer = document.createElement('div');
    jobTitleContainer.className = 'job-title';
    const headingElement = document.createElement('h2');
    headingElement.textContent = 'Lead Infrastructure Engineer';
    jobTitleContainer.appendChild(headingElement);
    document.body.appendChild(jobTitleContainer);

    const companyElement = document.createElement('div');
    companyElement.setAttribute('data-company', 'Acme Systems');
    document.body.appendChild(companyElement);

    const formElement = document.createElement('form');
    formElement.id = 'application-form';
    document.body.appendChild(formElement);

    const metadata = adapter.getJobMetadata();

    expect(metadata).not.toBeNull();
    expect(metadata?.jobTitle).toBe('Lead Infrastructure Engineer');
    expect(metadata?.companyName).toBe('Acme Systems');
    expect(metadata?.isApplicationPage).toBe(true);
  });
});
