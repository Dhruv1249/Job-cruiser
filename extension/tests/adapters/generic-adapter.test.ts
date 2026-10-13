import { beforeEach, describe, expect, it } from 'vitest';
import { GenericAdapter } from '../../lib/adapters/generic-adapter';
import { FieldClassification } from '../../lib/types';

describe('GenericAdapter', () => {
  let adapter: GenericAdapter;

  beforeEach(() => {
    adapter = new GenericAdapter();
    document.body.innerHTML = '';
    document.title = '';
    const happyDomWindow = window as unknown as {
      happyDOM: { setURL: (destinationUrl: string) => void };
    };
    happyDomWindow.happyDOM.setURL('http://localhost:3000/jobs/1');
  });

  it('always returns true for detect', () => {
    expect(adapter.detect()).toBe(true);
  });

  it('returns the same classification without modification in enhanceClassification', () => {
    const inputElement = document.createElement('input');
    const baseClassification: FieldClassification = {
      fieldType: 'email',
      confidence: 0.75,
      visibleLabel: 'Email Address',
      inputKind: 'text',
    };

    const enhanced = adapter.enhanceClassification(inputElement, baseClassification);

    expect(enhanced).toEqual(baseClassification);
  });

  it('extracts job title from h1 in getJobMetadata', () => {
    const headingElement = document.createElement('h1');
    headingElement.textContent = 'Full Stack Developer';
    document.body.appendChild(headingElement);

    const formElement = document.createElement('form');
    const inputElement = document.createElement('input');
    inputElement.type = 'text';
    formElement.appendChild(inputElement);
    document.body.appendChild(formElement);

    const metadata = adapter.getJobMetadata();

    expect(metadata).not.toBeNull();
    expect(metadata?.jobTitle).toBe('Full Stack Developer');
    expect(metadata?.isApplicationPage).toBe(true);
  });

  it('falls back to document.title when h1 is absent', () => {
    document.title = 'Staff Software Engineer Career Opportunity';

    const metadata = adapter.getJobMetadata();

    expect(metadata).not.toBeNull();
    expect(metadata?.jobTitle).toBe('Staff Software Engineer Career Opportunity');
  });
});
