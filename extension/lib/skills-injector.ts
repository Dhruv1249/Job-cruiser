import { InputInjector } from './input-injector';
import { SkillsInjectionMode, SkillsInjectionResult } from './types';

export type { SkillsInjectionMode, SkillsInjectionResult };

/**
 * Service managing automated injection of candidate skills into ATS tag lists and comboboxes.
 * Supports rapid comma-delimited dumping and incremental one-by-one cycling with dropdown selection.
 */
export class SkillsInjector {
  private readonly inputInjector: InputInjector;

  /**
   * Initializes a new instance of SkillsInjector.
   *
   * @param inputInjector - Optional custom instance of InputInjector for form interaction.
   */
  public constructor(inputInjector: InputInjector = new InputInjector()) {
    this.inputInjector = inputInjector;
  }

  /**
   * Injects skills sequentially one-by-one by typing into combobox inputs and selecting matching options.
   * Falls back to pressing the Enter key when no dropdown option appears.
   *
   * @param targetElement - The combobox container or text input element.
   * @param skills - Array of skill names to inject.
   * @returns Detailed outcome containing injected count, skipped count, and injection mode.
   */
  public async cycleOneByOne(
    targetElement: HTMLElement,
    skills: string[]
  ): Promise<SkillsInjectionResult> {
    let injectedCount = 0;
    let skippedCount = 0;

    for (const skill of skills) {
      const inputElement = this.findComboboxInput(targetElement);
      if (!inputElement) {
        skippedCount += 1;
        continue;
      }

      inputElement.focus();

      const propertyDescriptor = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        'value'
      );
      const nativePropertySetter = propertyDescriptor?.set;

      if (nativePropertySetter) {
        nativePropertySetter.call(inputElement, skill);
      } else {
        inputElement.value = skill;
      }

      inputElement.dispatchEvent(
        new Event('input', { bubbles: true, composed: true })
      );

      const matchingOption = await this.waitForDropdownOption(skill, 600);
      if (matchingOption) {
        (matchingOption as HTMLElement).click();
        injectedCount += 1;
      } else {
        inputElement.dispatchEvent(
          new KeyboardEvent('keydown', {
            key: 'Enter',
            code: 'Enter',
            bubbles: true,
            cancelable: true,
          })
        );
        skippedCount += 1;
      }

      const randomInterSkillDelay =
        Math.floor(Math.random() * (250 - 120 + 1)) + 120;
      await this.sleep(randomInterSkillDelay);
    }

    return {
      injectedCount,
      skippedCount,
      mode: 'cycle',
    };
  }

  /**
   * Injects all skills simultaneously as a comma-separated list into a text input or textarea.
   *
   * @param targetElement - The text input or textarea element receiving the concatenated list.
   * @param skills - Array of skill names to concatenate and inject.
   * @returns Detailed outcome containing injected count, skipped count, and injection mode.
   */
  public dumpAll(
    targetElement: HTMLElement,
    skills: string[]
  ): SkillsInjectionResult {
    if (skills.length === 0) {
      return {
        injectedCount: 0,
        skippedCount: 0,
        mode: 'dump',
      };
    }

    let inputElement: HTMLInputElement | HTMLTextAreaElement | null = null;
    const tagName = targetElement.tagName.toLowerCase();

    if (tagName === 'input' || tagName === 'textarea') {
      inputElement = targetElement as HTMLInputElement | HTMLTextAreaElement;
    } else {
      inputElement = targetElement.querySelector(
        'input, textarea'
      ) as HTMLInputElement | HTMLTextAreaElement | null;
    }

    if (inputElement) {
      const concatenatedSkills = skills.join(', ');
      this.inputInjector.injectTextValue(inputElement, concatenatedSkills);
      return {
        injectedCount: skills.length,
        skippedCount: 0,
        mode: 'dump',
      };
    }

    return {
      injectedCount: 0,
      skippedCount: skills.length,
      mode: 'dump',
    };
  }

  private findComboboxInput(container: HTMLElement): HTMLInputElement | null {
    if (container.tagName.toLowerCase() === 'input') {
      return container as HTMLInputElement;
    }

    const comboboxRoleInput = container.querySelector(
      'input[role="combobox"]'
    );
    if (comboboxRoleInput) {
      return comboboxRoleInput as HTMLInputElement;
    }

    const genericInput = container.querySelector('input');
    if (genericInput) {
      return genericInput as HTMLInputElement;
    }

    return null;
  }

  private async waitForDropdownOption(
    skillText: string,
    timeoutMs: number
  ): Promise<Element | null> {
    const normalizedSkill = skillText.trim().toLowerCase();
    const startTime = Date.now();
    const pollIntervalMilliseconds = 50;

    while (Date.now() - startTime <= timeoutMs) {
      const candidateElements = Array.from(
        document.querySelectorAll(
          '[role="option"], [role="listbox"] li, .dropdown-item, .select-option, li[data-value]'
        )
      );

      const matchingElement = candidateElements.find((candidate) => {
        const candidateText = candidate.textContent?.trim().toLowerCase() ?? '';
        return (
          candidateText === normalizedSkill ||
          candidateText.includes(normalizedSkill)
        );
      });

      if (matchingElement) {
        return matchingElement;
      }

      await this.sleep(pollIntervalMilliseconds);
    }

    return null;
  }

  private sleep(milliseconds: number): Promise<void> {
    return new Promise((resolve) => {
      setTimeout(resolve, milliseconds);
    });
  }
}
