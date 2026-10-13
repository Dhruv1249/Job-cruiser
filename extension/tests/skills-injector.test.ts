import { beforeEach, describe, expect, it, vi } from 'vitest';
import { InputInjector } from '../lib/input-injector';
import { SkillsInjector } from '../lib/skills-injector';

describe('SkillsInjector', () => {
  let injector: SkillsInjector;
  let mockInputInjector: InputInjector;

  beforeEach(() => {
    mockInputInjector = new InputInjector();
    injector = new SkillsInjector(mockInputInjector);
    document.body.innerHTML = '';
  });

  it('sets comma-separated value on text input when dumpAll is invoked', () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const skills = ['TypeScript', 'Preact', 'TailwindCSS'];
    const injectionResult = injector.dumpAll(inputElement, skills);

    expect(injectionResult.mode).toBe('dump');
    expect(injectionResult.injectedCount).toBe(3);
    expect(injectionResult.skippedCount).toBe(0);
    expect(inputElement.value).toBe('TypeScript, Preact, TailwindCSS');
  });

  it('sets comma-separated value on textarea when dumpAll is invoked', () => {
    const textareaElement = document.createElement('textarea');
    document.body.appendChild(textareaElement);

    const skills = ['Python', 'PostgreSQL', 'Docker'];
    const injectionResult = injector.dumpAll(textareaElement, skills);

    expect(injectionResult.mode).toBe('dump');
    expect(injectionResult.injectedCount).toBe(3);
    expect(injectionResult.skippedCount).toBe(0);
    expect(textareaElement.value).toBe('Python, PostgreSQL, Docker');
  });

  it('iterates and dispatches events for each skill during cycleOneByOne', async () => {
    const containerElement = document.createElement('div');
    const inputElement = document.createElement('input');
    inputElement.setAttribute('role', 'combobox');
    containerElement.appendChild(inputElement);
    document.body.appendChild(containerElement);

    const capturedInputValues: string[] = [];
    inputElement.addEventListener('input', () => {
      capturedInputValues.push(inputElement.value);

      const existingOption = document.querySelector('[role="option"]');
      if (existingOption) {
        existingOption.remove();
      }

      const optionElement = document.createElement('div');
      optionElement.setAttribute('role', 'option');
      optionElement.textContent = inputElement.value;
      document.body.appendChild(optionElement);
    });

    const skills = ['TypeScript', 'Rust'];
    const injectionResult = await injector.cycleOneByOne(
      containerElement,
      skills
    );

    expect(injectionResult.mode).toBe('cycle');
    expect(injectionResult.injectedCount).toBe(2);
    expect(injectionResult.skippedCount).toBe(0);
    expect(capturedInputValues).toEqual(['TypeScript', 'Rust']);
  });

  it('skips skills not found in dropdown after timeout and dispatches Enter key', async () => {
    const inputElement = document.createElement('input');
    document.body.appendChild(inputElement);

    const keyboardEventListener = vi.fn();
    inputElement.addEventListener('keydown', keyboardEventListener);

    const obscureSkills = ['ObscureProprietaryLanguage'];
    const injectionResult = await injector.cycleOneByOne(
      inputElement,
      obscureSkills
    );

    expect(injectionResult.mode).toBe('cycle');
    expect(injectionResult.injectedCount).toBe(0);
    expect(injectionResult.skippedCount).toBe(1);
    expect(keyboardEventListener).toHaveBeenCalledTimes(1);

    const dispatchedKeyboardEvent = keyboardEventListener.mock.calls[0]?.[0] as KeyboardEvent;
    expect(dispatchedKeyboardEvent.key).toBe('Enter');
  });
});
