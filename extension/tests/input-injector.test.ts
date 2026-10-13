import { beforeEach, describe, expect, it, vi } from 'vitest';
import { InputInjector } from '../lib/input-injector';

describe('InputInjector', () => {
  let injector: InputInjector;

  beforeEach(() => {
    injector = new InputInjector();
    document.body.innerHTML = '';
  });

  it('injects text value into text input and dispatches input and change events', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const inputEventListener = vi.fn();
    const changeEventListener = vi.fn();
    inputElement.addEventListener('input', inputEventListener);
    inputElement.addEventListener('change', changeEventListener);

    injector.injectTextValue(inputElement, 'Ada Lovelace');

    expect(inputElement.value).toBe('Ada Lovelace');
    expect(inputEventListener).toHaveBeenCalledTimes(1);
    expect(changeEventListener).toHaveBeenCalledTimes(1);
  });

  it('injects text value into textarea element and dispatches events', () => {
    const textareaElement = document.createElement('textarea');
    document.body.appendChild(textareaElement);

    const inputEventListener = vi.fn();
    const changeEventListener = vi.fn();
    textareaElement.addEventListener('input', inputEventListener);
    textareaElement.addEventListener('change', changeEventListener);

    injector.injectTextValue(
      textareaElement,
      'Passionate compiler engineer with ten years experience.'
    );

    expect(textareaElement.value).toBe(
      'Passionate compiler engineer with ten years experience.'
    );
    expect(inputEventListener).toHaveBeenCalledTimes(1);
    expect(changeEventListener).toHaveBeenCalledTimes(1);
  });

  it('finds and selects matching select option by exact value', () => {
    const selectElement = document.createElement('select');
    const defaultOption = document.createElement('option');
    defaultOption.value = '';
    defaultOption.textContent = 'Please choose an option';
    const firstOption = document.createElement('option');
    firstOption.value = 'yes';
    firstOption.textContent = 'Authorized to work';
    const secondOption = document.createElement('option');
    secondOption.value = 'no';
    secondOption.textContent = 'Requires sponsorship';

    selectElement.appendChild(defaultOption);
    selectElement.appendChild(firstOption);
    selectElement.appendChild(secondOption);
    document.body.appendChild(selectElement);

    const changeEventListener = vi.fn();
    selectElement.addEventListener('change', changeEventListener);

    const matchOutcome = injector.injectSelectValue(selectElement, 'yes');

    expect(matchOutcome).toBe(true);
    expect(selectElement.value).toBe('yes');
    expect(firstOption.selected).toBe(true);
    expect(changeEventListener).toHaveBeenCalledTimes(1);
  });

  it('finds and selects matching select option by partial text content', () => {
    const selectElement = document.createElement('select');
    const firstOption = document.createElement('option');
    firstOption.value = 'OPT_1';
    firstOption.textContent = 'Full-Time Permanent Employee';
    const secondOption = document.createElement('option');
    secondOption.value = 'OPT_2';
    secondOption.textContent = 'Part-Time Contractor';

    selectElement.appendChild(firstOption);
    selectElement.appendChild(secondOption);
    document.body.appendChild(selectElement);

    const matchOutcome = injector.injectSelectValue(selectElement, 'Permanent');

    expect(matchOutcome).toBe(true);
    expect(selectElement.value).toBe('OPT_1');
    expect(firstOption.selected).toBe(true);
  });

  it('returns false when no matching option exists in select element', () => {
    const selectElement = document.createElement('select');
    const optionElement = document.createElement('option');
    optionElement.value = 'engineering';
    optionElement.textContent = 'Engineering';
    selectElement.appendChild(optionElement);
    document.body.appendChild(selectElement);

    const matchOutcome = injector.injectSelectValue(selectElement, 'Marketing');

    expect(matchOutcome).toBe(false);
    expect(selectElement.value).toBe('engineering');
  });

  it('injects file upload with File and DataTransfer to file input element', async () => {
    const fileInputElement = document.createElement('input');
    fileInputElement.type = 'file';
    document.body.appendChild(fileInputElement);

    const changeEventListener = vi.fn();
    fileInputElement.addEventListener('change', changeEventListener);

    const documentBlob = new Blob(['Mock PDF Resume Content'], {
      type: 'application/pdf',
    });

    await injector.injectFileUpload(
      fileInputElement,
      documentBlob,
      'candidate_resume.pdf'
    );

    expect(fileInputElement.files).not.toBeNull();
    expect(fileInputElement.files?.length).toBe(1);
    const uploadedFile = fileInputElement.files?.[0];
    expect(uploadedFile?.name).toBe('candidate_resume.pdf');
    expect(uploadedFile?.type).toBe('application/pdf');
    expect(changeEventListener).toHaveBeenCalledTimes(1);
  });
});
